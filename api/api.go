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
	mes := periodo.Format("January")
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

	camposAdicionales := []CampoAdicionalModel{
		{Clave: "numeroMedidor", Valor: "0"},
		{Clave: "mes", Valor: mes},
		{Clave: "gestion", Valor: gestion},
		{Clave: "ciudad", Valor: "Tupiza"},
		{Clave: "zona", Valor: zona},
		{Clave: "domicilioCliente", Valor: ifEmpty(calle, "Sin dirección")},
		{Clave: "consumoPeriodo", Valor: fmt.Sprintf("%f", con_m3)},
		{Clave: "ajusteSujetoIva", Valor: fmt.Sprintf("%f", ajusteSejetoIvaTotal)},
		{Clave: "detalleAjusteSujetoIva", Valor: fmt.Sprintf("%v", ajusteSejetoIvaDetalleJson)},
	}

	if desc_ley1886 > 0 {
		camposAdicionales = append(camposAdicionales, CampoAdicionalModel{Clave: "beneficiarioLey1886", Valor: nit})
	}

	solicitud := SolicitudModel{
		CodigoModalidad: 1,
		CodigoEmision: 1,
		CodigoDocumentoSector: 13,
		CodigoSucursal: 0,
		CodigoAmbiente: 1,
		CodigoPuntoVenta: 0,
		CodigoActividad: 360000,
		NitEmisor: "1023807025",
		CodigoTipoEvento: 0,
		Leyenda: "Leyenda",
		NumeroDocumento: nit,
		CodigoTipoDocumento: 1,
		ComplementoDocumento: "",
		RazonSocial: razon,
		CorreoCliente: "",
		FechaEmision: fechaHora,
	}

	cabecera := CabeceraModel{
		NitEmisor: 1023807025,
		RazonSocialEmisor: "EMPSAAT",
		Municipio: "TUPIZA",
		Telefono: "(2) 6944636",
		CodigoSucursal: 0,
		Direccion: "Calle Bolivar S/N Zona central",
		CodigoPuntoVenta: 0,
		NombreRazonSocial: razon,
		CodigoTipoDocumentoIdentidad: obtenerTipoDocumento(nit),
		NumeroDocumento: nit,
		Complemento: "",
		CodigoCliente: abonado,
		CodigoMetodoPago: 1,
		NumeroTarjeta: 0,
		MontoTotal: impFactura,
		MontoTotalSujetoIva: impFactura,
		CodigoMoneda: 1,
		TipoCambio: 1,
		MontoTotalMoneda: impFactura,
		MontoGiftCard: 0,
		DescuentoAdicional: 0,
		CodigoExcepcion: 1,
		Cafc: "",
		Leyenda: "hola",
		Usuario: "Santiago",
		CodigoDocumentoSector: 13,
		FechaEmision: fechaHora,
		CamposAdicionales: camposAdicionales,
	}

	if numero > 0 {
		solicitud.NumeroFactura = numero
		cabecera.NumeroFactura = numero
	}

	detalle := []DetalleModel{
		{
			ActividadEconomica: 360000,
			CodigoProductoSin: 86330,
			CodigoProducto: "001",
			Descripcion: "SUBTOTAL SERVICIO DE AGUA",
			Cantidad: 1,
			UnidadMedida: 58,
			PrecioUnitario: impTotal + descuento,
			MontoDescuento: descuento,
			SubTotal: impTotal,
			CamposAdicionales: []CampoAdicionalModel{},
		},
	}

	facturaRequest := FacturaRequest{
		Solicitud: solicitud,
		Cabecera: cabecera,
		Detalle: detalle,
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
		CodigoModalidad: 1,
		CodigoEmision: 1,
		CodigoDocumentoSector: 1,
		CodigoSucursal: 0,
		CodigoAmbiente: 1,
		CodigoPuntoVenta: 0,
		CodigoActividad: 360000,
		NitEmisor: "1023807025",
		CodigoTipoEvento: 0,
		Leyenda: "Leyenda",
		NumeroDocumento: nit,
		CodigoTipoDocumento: 1,
		ComplementoDocumento: "",
		RazonSocial: razon,
		CorreoCliente: "",
		FechaEmision: fechaHora,
	}

	cabecera := CabeceraModel{
		NitEmisor: 1023807025,
		RazonSocialEmisor: "EMPSAAT",
		Municipio: "TUPIZA",
		Telefono: "(2) 6944636",
		CodigoSucursal: 0,
		Direccion: "Calle Bolivar S/N Zona central",
		CodigoPuntoVenta: 0,
		NombreRazonSocial: razon,
		CodigoTipoDocumentoIdentidad: obtenerTipoDocumento(nit),
		NumeroDocumento: nit,
		Complemento: "",
		CodigoCliente: abonado,
		CodigoMetodoPago: 1,
		NumeroTarjeta: 0,
		MontoTotal: impTotal,
		MontoTotalSujetoIva: impTotal,
		CodigoMoneda: 1,
		TipoCambio: 1,
		MontoTotalMoneda: impTotal,
		MontoGiftCard: 0,
		DescuentoAdicional: 0,
		CodigoExcepcion: 1,
		Cafc: "",
		Leyenda: "hola",
		Usuario: "Santiago",
		CodigoDocumentoSector: 1,
		FechaEmision: fechaHora,
		CamposAdicionales: []CampoAdicionalModel{},
	}

	detalle := []DetalleModel{}
	for _, item := range facturaDetalle {
		detalle = append(detalle, DetalleModel{
			ActividadEconomica: 360000,
			CodigoProductoSin: 86330,
			CodigoProducto: item.CodigoProducto,
			Descripcion: item.Descripcion,
			Cantidad: item.Cantidad,
			UnidadMedida: 62,
			PrecioUnitario: item.PrecioUnitario,
			MontoDescuento: 0,
			SubTotal: item.SubTotal,
			CamposAdicionales: []CampoAdicionalModel{},
		})
	}

	facturaRequest := FacturaRequest{
		Solicitud: solicitud,
		Cabecera: cabecera,
		Detalle: detalle,
		ExtraInfo: []ExtraInfoModel{},
	}

	return fe.sendFacturaRequest(facturaRequest)
}

func (fe *FacturacionElectronica) sendFacturaRequest(facturaRequest FacturaRequest) (map[string]interface{}, error) {
	jsonData, err := json.MarshalIndent(facturaRequest, "", "  ")
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
	CodigoProducto string
	Descripcion    string
	Cantidad       int
	UnidadMedida   int
	PrecioUnitario float64
	MontoDescuento float64
	SubTotal       float64
}

type FacturaRequest struct {
	Solicitud SolicitudModel
	Cabecera  CabeceraModel
	Detalle   []DetalleModel
	ExtraInfo []ExtraInfoModel
}

type SolicitudModel struct {
	CodigoModalidad        int
	CodigoEmision          int
	CodigoDocumentoSector  int
	CodigoSucursal         int
	CodigoAmbiente         int
	CodigoPuntoVenta       int
	CodigoActividad        int
	NitEmisor              string
	CodigoTipoEvento       int
	Leyenda                string
	NumeroDocumento        string
	CodigoTipoDocumento    int
	ComplementoDocumento   string
	RazonSocial            string
	CorreoCliente          string
	FechaEmision           string
	NumeroFactura          int
	FormatoPdf             string
}

type CabeceraModel struct {
	NitEmisor                int64
	RazonSocialEmisor        string
	Municipio                string
	Telefono                 string
	NumeroFactura            int
	Cuf                      string
	Cufd                     string
	CodigoSucursal           int
	Direccion                string
	CodigoPuntoVenta         int
	FechaEmision             string
	NombreRazonSocial        string
	CodigoTipoDocumentoIdentidad int
	NumeroDocumento          string
	Complemento              string
	CodigoCliente            string
	CodigoMetodoPago         int
	NumeroTarjeta            int
	MontoTotal               float64
	MontoTotalSujetoIva      float64
	CodigoMoneda             int
	TipoCambio               float64
	MontoTotalMoneda         float64
	MontoGiftCard            float64
	DescuentoAdicional       float64
	CodigoExcepcion          int
	Cafc                     string
	Leyenda                  string
	Usuario                  string
	CodigoDocumentoSector    int
	CamposAdicionales        []CampoAdicionalModel
}

type CampoAdicionalModel struct {
	Clave  string
	Valor  string
}

type DetalleModel struct {
	ActividadEconomica int
	CodigoProductoSin  int
	CodigoProducto     string
	Descripcion        string
	Cantidad           int
	UnidadMedida       int
	PrecioUnitario     float64
	MontoDescuento     float64
	SubTotal           float64
	CamposAdicionales  []CampoAdicionalModel
}

type ExtraInfoModel struct {
	Key   string
	Value string
	Label string
}