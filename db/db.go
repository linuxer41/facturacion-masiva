package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/denisenkom/go-mssqldb"
)

var DB *sql.DB

type Factura struct {
    Abonado     string
    Lectura     int
    ConM3       float64
    LecEstimada bool // int
    ImpFijo     float64
    ImpAdic     float64
    ImpTotal    float64
    ImpAlcanta  float64
    ImpRep      float64
    ImpRecargo  float64
    ImpFactura  float64
    ImpLey1886  float64
    FecPago     *string 
    FacturaID   int
    NumFactura  int
    NODOC       string
    Categoria   string
    Zona        string
    Calle       string
    Ley1886     string
    Nit         string
    Razon       string
    Liberacion  string
}

func InitDB(connString string) {
	var err error
	DB, err = sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal(err)
	}
}

func GetFactores() ([]map[string]interface{}, error) {
	query := `SELECT * FROM Factores WHERE Estado = 1 AND Proceso = 1`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseRows(rows)
}

func VerificarLecturasFaltantes(emision string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT ABONADO FROM USUARIOS 
		WHERE ESTADO = 'N' 
		AND ABONADO NOT IN (
			SELECT ABONADO FROM FACTURAS 
			WHERE CONVERT(date, Emision) = '%s' 
			AND SERVICIO = 1
		)`, emision)
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseRows(rows)
}

func parseRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	var result []map[string]interface{}

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		row := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			row[col] = v
		}
		result = append(result, row)
	}

	return result, nil
}

func UpdateFacturaNumero(emision string) error {
	query := fmt.Sprintf(`
		UPDATE Facturas
		SET Num_Factura = factura - 1707433
		WHERE Num_Factura = 0 AND CONVERT(date, Emision) = '%s'
		
	`, emision)
	_, err := DB.Exec(query)
	return err
}

// func GetFacturasParaFacturacion(emision string) ([]map[string]interface{}, error) {
// 	query := fmt.Sprintf(`
// 		SELECT
// 			facturas.abonado, 
// 			facturas.lectura,
// 			facturas.con_m3,
// 			facturas.lec_estimada,
// 			facturas.Imp_Fijo,
// 			facturas.Imp_Adic,
// 			facturas.Imp_Total,
// 			facturas.Imp_Alcanta,
// 			facturas.Imp_Rep,
// 			facturas.Imp_Recargo,
// 			facturas.Imp_Factura,
// 			COALESCE(facturas.imp_ley1886_1 + facturas.imp_ley1886_2, 0) as imp_ley1886,
// 			facturas.Fec_Pago,
// 			facturas.Factura,
// 			facturas.Num_Factura,
// 			Usuarios.NODOC,
// 			Usuarios.Categoria,
// 			Usuarios.zona,
// 			Usuarios.calle,
// 			Usuarios.ley1886,
// 			CLIENTE.Nit,
// 			CLIENTE.RAZON,
// 			Usuarios.Liberacion
// 		FROM facturas
// 		LEFT JOIN Usuarios ON Usuarios.Abonado = facturas.abonado  
// 		LEFT JOIN CLIENTE ON CLIENTE.CLIENTE = Usuarios.NODOC
// 		WHERE CONVERT(date, facturas.emision) = '%s'
// 		AND facturas.servicio = 1 
// 		AND facturas.imp_factura > 0 
// 		AND facturas.Codigo_control IS NULL 
// 		ORDER BY facturas.num_factura ASC
// 	`, emision)
// 	rows, err := DB.Query(query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	return parseRows(rows)
// }

func GetFacturasParaFacturacion(emision string) ([]Factura, error) {
    query := `SELECT
        facturas.abonado, 
        facturas.lectura,
        facturas.con_m3,
        facturas.lec_estimada,
        facturas.Imp_Fijo,
        facturas.Imp_Adic,
        facturas.Imp_Total,
        facturas.Imp_Alcanta,
        facturas.Imp_Rep,
        facturas.Imp_Recargo,
        facturas.Imp_Factura,
        COALESCE(facturas.imp_ley1886_1 + facturas.imp_ley1886_2, 0) as imp_ley1886,
        facturas.Fec_Pago,
        facturas.Factura,
        facturas.Num_Factura,
        Usuarios.NODOC,
        Usuarios.Categoria,
        Usuarios.zona,
        Usuarios.calle,
        Usuarios.ley1886,
        CLIENTE.Nit,
        CLIENTE.RAZON,
        Usuarios.Liberacion
    FROM facturas
    LEFT JOIN Usuarios ON Usuarios.Abonado = facturas.abonado  
    LEFT JOIN CLIENTE ON CLIENTE.CLIENTE = Usuarios.NODOC
    WHERE CONVERT(date, facturas.emision) = @p1
    AND facturas.servicio = 1 
    AND facturas.imp_factura > 0 
    AND facturas.Codigo_control IS NULL 
    ORDER BY facturas.num_factura ASC`

    rows, err := DB.Query(query, emision)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var facturas []Factura
    for rows.Next() {
        var f Factura
        var fecPago sql.NullString  // Usar sql.NullString para manejar NULL
        err := rows.Scan(
            &f.Abonado, &f.Lectura, &f.ConM3, &f.LecEstimada,
            &f.ImpFijo, &f.ImpAdic, &f.ImpTotal, &f.ImpAlcanta,
            &f.ImpRep, &f.ImpRecargo, &f.ImpFactura, &f.ImpLey1886,
            &fecPago, &f.FacturaID, &f.NumFactura, &f.NODOC,
            &f.Categoria, &f.Zona, &f.Calle, &f.Ley1886,
            &f.Nit, &f.Razon, &f.Liberacion,
        )
        if err != nil {
            return nil, err
        }
        if fecPago.Valid {
            f.FecPago = &fecPago.String
        } else {
            f.FecPago = nil
        }
        facturas = append(facturas, f)
    }
    return facturas, nil
}

func UpdateFacturaCodigoControl(factura int, codigoControl string) error {
	query := fmt.Sprintf(`
		UPDATE Facturas 
		SET Codigo_Control = '%s' 
		WHERE Factura = %d
	`, codigoControl, factura)
	_, err := DB.Exec(query)
	return err
}