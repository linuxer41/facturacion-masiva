package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/goodsign/monday"
)

type ApiConfig struct {
	Url    string
	ApiKey string
}

type FacturacionElectronica struct {
	client  *http.Client
	baseUrl string
	apiKey  string
}

// func round(val float64, precision int) float64 {
// 	ratio := math.Pow(10, float64(precision))
// 	return math.Round(val*ratio) / ratio
// }

func NewFacturacionElectronica(apiConfig ApiConfig) *FacturacionElectronica {
	return &FacturacionElectronica{
		client:  &http.Client{},
		baseUrl: apiConfig.Url,
		apiKey:  apiConfig.ApiKey,
	}
}

func (fe *FacturacionElectronica) FacturaServicios(
	periodo time.Time,
	con_m3, impTotal, impAlcanta, impRep, impFactura, impRecargo, desc_ley1886 float64,
	razon, abonado, nit, zona, calle string,
	numero int,
) (map[string]interface{}, error) {
	// mes := periodo.Format("January")
	mes := monday.Format(periodo, "January", monday.LocaleEsES)
	gestion := periodo.Format("2006")

	nit = ifZero(nit, abonado)

	descuento := 0.0
	if desc_ley1886 > 0 {
		descuento = desc_ley1886
		impFactura = impFactura
		impTotal = impTotal
	}

	fechaHora := time.Now().Format("2006-01-02T15:04:05.000")

	ajusteSejetoIvaTotal := round(impAlcanta + impRep + impRecargo)

	ajusteSejetoIvaDetalleJson := map[string]interface{}{}
	if impAlcanta > 0 {
		ajusteSejetoIvaDetalleJson["Alcantarillado"] = fmt.Sprintf("%f", impAlcanta)
	}
	if impRep > 0 {
		ajusteSejetoIvaDetalleJson["Rep. Formulario"] = fmt.Sprintf("%f", impRep)
	}
	if impRecargo > 0 {
		ajusteSejetoIvaDetalleJson["Recargo"] = fmt.Sprintf("%f", impRecargo)
	}

	// Convertir el mapa a JSON
	ajusteSejetoIvaDetalleJsonString, err := json.Marshal(ajusteSejetoIvaDetalleJson)
	if err != nil {
		return nil, fmt.Errorf("error al convertir ajusteSejetoIvaDetalleJson a JSON: %v", err)
	}

	camposAdicionales := []CampoAdicionalModel{
		{Clave: "numeroMedidor", Valor: "0"},
		{Clave: "mes", Valor: mes},
		{Clave: "gestion", Valor: gestion},
		{Clave: "ciudad", Valor: "Tupiza"},
		{Clave: "zona", Valor: zona},
		{Clave: "domicilioCliente", Valor: ifEmpty(calle, "Sin dirección")},
		{Clave: "consumoPeriodo", Valor: fmt.Sprintf("%.2f", con_m3)},
		{Clave: "ajusteSujetoIva", Valor: fmt.Sprintf("%.2f", ajusteSejetoIvaTotal)},
		{Clave: "detalleAjusteSujetoIva", Valor: string(ajusteSejetoIvaDetalleJsonString)},
	}

	if desc_ley1886 > 0 {
		camposAdicionales = append(camposAdicionales, CampoAdicionalModel{Clave: "beneficiarioLey1886", Valor: nit})
	}

	solicitud := SolicitudModel{
		CodigoModalidad:       1,
		CodigoEmision:         1,
		CodigoDocumentoSector: 13,
		CodigoSucursal:        0,
		CodigoAmbiente:        1,
		CodigoPuntoVenta:      0,
		CodigoActividad:       360000,
		NitEmisor:             "1023807025",
		CodigoTipoEvento:      0,
		// Leyenda: "Leyenda",
		// NumeroDocumento: nit,
		// CodigoTipoDocumento: 1,
		// ComplementoDocumento: "",
		// RazonSocial: razon,
		// CorreoCliente: "",
		FechaEmision: fechaHora,
		FormatoPdf:   1,
	}

	cabecera := CabeceraModel{
		NitEmisor:                    1023807025,
		RazonSocialEmisor:            "EMPSAAT",
		Municipio:                    "TUPIZA",
		Telefono:                     "(2) 6944636",
		CodigoSucursal:               0,
		Direccion:                    "Calle Bolivar S/N Zona central",
		CodigoPuntoVenta:             0,
		NombreRazonSocial:            razon,
		CodigoTipoDocumentoIdentidad: obtenerTipoDocumento(nit),
		NumeroDocumento:              nit,
		Complemento:                  "",
		CodigoCliente:                abonado,
		CodigoMetodoPago:             1,
		NumeroTarjeta:                0,
		MontoTotal:                   impFactura,
		MontoTotalSujetoIva:          impFactura,
		CodigoMoneda:                 1,
		TipoCambio:                   1,
		MontoTotalMoneda:             impFactura,
		MontoGiftCard:                0,
		DescuentoAdicional:           0,
		CodigoExcepcion:              1,
		Cafc:                         "",
		Leyenda:                      "hola",
		Usuario:                      "Santiago",
		CodigoDocumentoSector:        13,
		FechaEmision:                 fechaHora,
		CamposAdicionales:            camposAdicionales,
	}

	if numero > 0 {
		solicitud.NumeroFactura = numero
		cabecera.NumeroFactura = numero
	}

	detalle := []DetalleModel{
		{
			ActividadEconomica: 360000,
			CodigoProductoSin:  86330,
			CodigoProducto:     "001",
			Descripcion:        "SUBTOTAL SERVICIO DE AGUA",
			Cantidad:           1,
			UnidadMedida:       58,
			PrecioUnitario:     round(impTotal + descuento),
			MontoDescuento:     descuento,
			SubTotal:           impTotal,
			CamposAdicionales:  []CampoAdicionalModel{},
		},
	}

	facturaRequest := FacturaRequest{
		Solicitud: solicitud,
		Cabecera:  cabecera,
		Detalle:   detalle,
		ExtraInfo: []ExtraInfoModel{},
	}

	return fe.sendFacturaRequest(facturaRequest)
}

func (fe *FacturacionElectronica) FacturaCompraVenta(
	facturaDetalle []FacturacionCompraVentaDetalle,
	impTotal float64,
	razon, abonado, nit string,
	numero int,
) (map[string]interface{}, error) {
	nit = ifZero(nit, abonado)

	fechaHora := time.Now().Format("2006-01-02T15:04:05.000")

	solicitud := SolicitudModel{
		CodigoModalidad:       1,
		CodigoEmision:         1,
		CodigoDocumentoSector: 1,
		CodigoSucursal:        0,
		CodigoAmbiente:        1,
		CodigoPuntoVenta:      0,
		CodigoActividad:       360000,
		NitEmisor:             "1023807025",
		CodigoTipoEvento:      0,
		Leyenda:               "Leyenda",
		// NumeroDocumento: nit,
		// CodigoTipoDocumento: null,
		// ComplementoDocumento: "",
		// RazonSocial: razon,
		// CorreoCliente: "",
		FechaEmision: fechaHora,
	}

	cabecera := CabeceraModel{
		NitEmisor:                    1023807025,
		RazonSocialEmisor:            "EMPSAAT",
		Municipio:                    "TUPIZA",
		Telefono:                     "(2) 6944636",
		CodigoSucursal:               0,
		Direccion:                    "Calle Bolivar S/N Zona central",
		CodigoPuntoVenta:             0,
		NombreRazonSocial:            razon,
		CodigoTipoDocumentoIdentidad: obtenerTipoDocumento(nit),
		NumeroDocumento:              nit,
		Complemento:                  "",
		CodigoCliente:                abonado,
		CodigoMetodoPago:             1,
		NumeroTarjeta:                0,
		MontoTotal:                   impTotal,
		MontoTotalSujetoIva:          impTotal,
		CodigoMoneda:                 1,
		TipoCambio:                   1,
		MontoTotalMoneda:             impTotal,
		MontoGiftCard:                0,
		DescuentoAdicional:           0,
		CodigoExcepcion:              1,
		Cafc:                         "",
		Leyenda:                      "hola",
		Usuario:                      "Santiago",
		CodigoDocumentoSector:        1,
		FechaEmision:                 fechaHora,
		CamposAdicionales:            []CampoAdicionalModel{},
	}

	detalle := []DetalleModel{}
	for _, item := range facturaDetalle {
		detalle = append(detalle, DetalleModel{
			ActividadEconomica: 360000,
			CodigoProductoSin:  86330,
			CodigoProducto:     item.CodigoProducto,
			Descripcion:        item.Descripcion,
			Cantidad:           item.Cantidad,
			UnidadMedida:       62,
			PrecioUnitario:     item.PrecioUnitario,
			MontoDescuento:     0,
			SubTotal:           item.SubTotal,
			CamposAdicionales:  []CampoAdicionalModel{},
		})
	}

	facturaRequest := FacturaRequest{
		Solicitud: solicitud,
		Cabecera:  cabecera,
		Detalle:   detalle,
		ExtraInfo: []ExtraInfoModel{},
	}

	return fe.sendFacturaRequest(facturaRequest)
}

func (fe *FacturacionElectronica) sendFacturaRequest(facturaRequest FacturaRequest) (map[string]interface{}, error) {
	jsonData, err := json.MarshalIndent(facturaRequest, "", "  ")
	// println("jsonData:", string(jsonData))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/invoice-utils/third-party-create", fe.baseUrl), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", fe.apiKey)

	resp, err := fe.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		println("jsonData:", string(jsonData))
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (fe *FacturacionElectronica) GetFile(cuf string, abonado int) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/invoice-utils/pdf?cuf=%s&formato=4", fe.baseUrl, cuf), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("api_key", fe.apiKey)

	resp, err := fe.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	tempFile, err := os.CreateTemp("./facturas", fmt.Sprintf("factura_abonado_%d_*.pdf", abonado))
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func obtenerTipoDocumento(documento string) int {
	if len(documento) > 8 {
		return 5
	} else if len(documento) > 6 {
		return 1
	} else {
		return 4
	}
}

func ifZero(value, fallback string) string {
	if value == "0" {
		return fallback
	}
	return value
}

func ifEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

// Definiciones de estructuras y tipos para la API
type FacturacionCompraVentaDetalle struct {
	CodigoProducto string  `json:"codigoProducto"`
	Descripcion    string  `json:"descripcion"`
	Cantidad       int     `json:"cantidad"`
	UnidadMedida   int     `json:"unidadMedida"`
	PrecioUnitario float64 `json:"precioUnitario"`
	MontoDescuento float64 `json:"montoDescuento"`
	SubTotal       float64 `json:"subTotal"`
}

type FacturaRequest struct {
	Solicitud SolicitudModel   `json:"solicitud"`
	Cabecera  CabeceraModel    `json:"cabecera"`
	Detalle   []DetalleModel   `json:"detalle"`
	ExtraInfo []ExtraInfoModel `json:"extraInfo"`
}

type SolicitudModel struct {
	CodigoModalidad       int    `json:"codigoModalidad"`
	CodigoEmision         int    `json:"codigoEmision"`
	CodigoDocumentoSector int    `json:"codigoDocumentoSector"`
	CodigoSucursal        int    `json:"codigoSucursal"`
	CodigoAmbiente        int    `json:"codigoAmbiente"`
	CodigoPuntoVenta      int    `json:"codigoPuntoVenta"`
	CodigoActividad       int    `json:"codigoActividad"`
	NitEmisor             string `json:"nitEmisor"`
	CodigoTipoEvento      int    `json:"codigoTipoEvento"`
	Leyenda               string `json:"leyenda,omitempty"`
	NumeroDocumento       string `json:"numeroDocumento,omitempty"`
	CodigoTipoDocumento   int    `json:"codigoTipoDocumento,omitempty"`
	ComplementoDocumento  string `json:"complementoDocumento,omitempty"`
	RazonSocial           string `json:"razonSocial,omitempty"`
	CorreoCliente         string `json:"correoCliente,omitempty"`
	FechaEmision          string `json:"fechaEmision,omitempty"`
	NumeroFactura         int    `json:"numeroFactura,omitempty"`
	FormatoPdf            int    `json:"formatoPdf"`
}

type CabeceraModel struct {
	NitEmisor                    int64                 `json:"nitEmisor"`
	RazonSocialEmisor            string                `json:"razonSocialEmisor"`
	Municipio                    string                `json:"municipio"`
	Telefono                     string                `json:"telefono"`
	NumeroFactura                int                   `json:"numeroFactura"`
	Cuf                          string                `json:"cuf"`
	Cufd                         string                `json:"cufd"`
	CodigoSucursal               int                   `json:"codigoSucursal"`
	Direccion                    string                `json:"direccion"`
	CodigoPuntoVenta             int                   `json:"codigoPuntoVenta"`
	FechaEmision                 string                `json:"fechaEmision"`
	NombreRazonSocial            string                `json:"nombreRazonSocial"`
	CodigoTipoDocumentoIdentidad int                   `json:"codigoTipoDocumentoIdentidad"`
	NumeroDocumento              string                `json:"numeroDocumento"`
	Complemento                  string                `json:"complemento"`
	CodigoCliente                string                `json:"codigoCliente"`
	CodigoMetodoPago             int                   `json:"codigoMetodoPago"`
	NumeroTarjeta                int                   `json:"numeroTarjeta"`
	MontoTotal                   float64               `json:"montoTotal"`
	MontoTotalSujetoIva          float64               `json:"montoTotalSujetoIva"`
	CodigoMoneda                 int                   `json:"codigoMoneda"`
	TipoCambio                   float64               `json:"tipoCambio"`
	MontoTotalMoneda             float64               `json:"montoTotalMoneda"`
	MontoGiftCard                float64               `json:"montoGiftCard"`
	DescuentoAdicional           float64               `json:"descuentoAdicional"`
	CodigoExcepcion              int                   `json:"codigoExcepcion"`
	Cafc                         string                `json:"cafc"`
	Leyenda                      string                `json:"leyenda"`
	Usuario                      string                `json:"usuario"`
	CodigoDocumentoSector        int                   `json:"codigoDocumentoSector"`
	CamposAdicionales            []CampoAdicionalModel `json:"camposAdicionales"`
}

type CampoAdicionalModel struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type DetalleModel struct {
	ActividadEconomica int                   `json:"actividadEconomica"`
	CodigoProductoSin  int                   `json:"codigoProductoSin"`
	CodigoProducto     string                `json:"codigoProducto"`
	Descripcion        string                `json:"descripcion"`
	Cantidad           int                   `json:"cantidad"`
	UnidadMedida       int                   `json:"unidadMedida"`
	PrecioUnitario     float64               `json:"precioUnitario"`
	MontoDescuento     float64               `json:"montoDescuento"`
	SubTotal           float64               `json:"subTotal"`
	CamposAdicionales  []CampoAdicionalModel `json:"camposAdicionales"`
}

type ExtraInfoModel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Label string `json:"label"`
}
