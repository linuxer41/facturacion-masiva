package ui

import (
	"fmt"
	"log"
	"app/db"
	"app/api"
	"sync"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func SetupUI() {
	a := app.New()
	w := a.NewWindow("Factura Processor")

	// Barra de pasos
	steps := container.NewVBox(
		widget.NewLabel("Cargar datos"),
		widget.NewLabel("Verificando datos"),
		widget.NewLabel("Emitiendo facturas"),
		widget.NewLabel("Finalizado"),
	)

	// Barra de progreso global
	globalProgressBar := widget.NewProgressBar()
	globalProgressLabel := widget.NewLabel("Progreso=> total: 0, procesados: 0, éxito: 0, error: 0")

	// Botón de inicio
	startButton := widget.NewButton("Start", func() {
		// startButton.Disable()
		go func() {
			// defer startButton.Enable()

			// Paso 1: Cargar los datos
			factores, err := db.GetFactores()
			if err != nil {
				log.Println("Error fetching factores:", err)
				return
			}
			if len(factores) == 0 {
				log.Println("No factores found")
				return
			}
			emision := factores[0]["Emision"].(time.Time).Format("2006-01-02")

			// Paso 2: Verificar los datos
			faltantes, err := db.VerificarLecturasFaltantes(emision)
			if err != nil {
				log.Println("Error verifying missing readings:", err)
				return
			}
			if len(faltantes) > 0 {
				log.Println("Lecturas faltantes:", faltantes)
				return
			}

			// Paso 3: Emitir facturas
			err = db.UpdateFacturaNumero(emision)
			if err != nil {
				log.Println("Error updating factura numbers:", err)
				return
			}

			facturas, err := db.GetFacturasParaFacturacion(emision)
			if err != nil {
				log.Println("Error fetching facturas for facturación:", err)
				return
			}

			var wg sync.WaitGroup
			totalFacturas := len(facturas)
			globalProgressBar.Max = float64(totalFacturas)
			procesados := 0
			exitos := 0
			fallos := 0

			for _, factura := range facturas {
				wg.Add(1)
				go func(factura map[string]interface{}) {
					defer wg.Done()
					defer func() {
						procesados++
						globalProgressBar.SetValue(float64(procesados))
						globalProgressLabel.SetText(fmt.Sprintf("Progreso=> total: %d, procesados: %d, éxito: %d, error: %d", totalFacturas, procesados, exitos, fallos))
					}()

					abonado := factura["abonado"].(string)
					// lectura := factura["lectura"].(int)
					con_m3 := factura["con_m3"].(float64)
					impTotal := factura["Imp_Total"].(float64)
					impAlcanta := factura["Imp_Alcanta"].(float64)
					impRep := factura["Imp_Rep"].(float64)
					impFactura := factura["Imp_Factura"].(float64)
					impRecargo := factura["Imp_Recargo"].(float64)
					desc_ley1886 := factura["imp_ley1886"].(float64)
					razon := factura["RAZON"].(string)
					nit := factura["Nit"].(string)
					zona := factura["zona"].(string)
					calle := factura["calle"].(string)
					num_Factura := factura["Num_Factura"].(int)
					facturaID := factura["Factura"].(int)

					fe := api.NewFacturacionElectronica(api.ApiConfig{
						Url:    "http://code.iathings.com:3001",
						ApiKey: "your_api_key_here",
					})

					result, err := fe.FacturaServicios(
						time.Now(), // Replace with actual emission date
						con_m3, impTotal, impAlcanta, impRep, impFactura, impRecargo, desc_ley1886,
						razon, abonado, nit, zona, calle, num_Factura,
					)
					if err != nil {
						log.Println("Error generating factura:", err)
						fallos++
						return
					}

					cuf := result["cuf"].(string)
					err = db.UpdateFacturaCodigoControl(facturaID, cuf)
					if err != nil {
						log.Println("Error updating factura codigo control:", err)
						fallos++
						return
					}

					exitos++
				}(factura)
			}

			wg.Wait()

			// Paso 4: Finalizado con éxito
			globalProgressLabel.SetText(fmt.Sprintf("Progreso=> total: %d, procesados: %d, éxito: %d, error: %d", totalFacturas, procesados, exitos, fallos))
		}()
	})

	// Combinar todos los elementos en un layout
	content := container.NewVBox(
		steps,
		globalProgressBar,
		globalProgressLabel,
		startButton,
	)

	w.SetContent(content)
	w.Resize(w.Content().MinSize())
	w.CenterOnScreen()
	w.ShowAndRun()
}