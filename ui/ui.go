package ui

import (
	"app/api"
	"app/db"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type AppState struct {
	CurrentView string
	Facturas    []db.Factura
	Emision     string
	ErrorMessage string
}

type C = layout.Context
type D = layout.Dimensions

// Define the enum for the status
type Status int

const (
	Pending Status = iota
	Processing
	Completed
)

// String representation of the Status enum
func (s Status) String() string {
	return [...]string{"Pending", "Processing", "Completed"}[s]
}

// Define the Step struct
type Step struct {
	name     string
	status   Status
	hasError bool
}

// Define the progress variables, a channel and a variable
var progressIncrementer chan bool

func SetupUI() {
	// Setup a separate channel to provide ticks to increment progress
	progressIncrementer = make(chan bool)
	go func() {
		for {
			time.Sleep(time.Second / 25)
			progressIncrementer <- true
		}
	}()

	// Create a new window for the loading screen
	loadingWindow := new(app.Window)
	loadingWindow.Option(app.Title("Facturación Masiva"))
	loadingWindow.Option(app.Size(unit.Dp(500), unit.Dp(300)))

	// Initialize the app state
	appState := &AppState{
		CurrentView: "main",
	}

	// Start the loading screen
	go func() {
		if err := drawLoadingScreen(loadingWindow, appState); err != nil {
			log.Fatal(err)
		}
	}()

	app.Main()
}

func drawLoadingScreen(w *app.Window, appState *AppState) error {
	// ops are the operations from the UI
	var ops op.Ops

	// th defines the material design style
	th := material.NewTheme()

	go func() {
		factores, err := db.GetFactores()
		if err != nil {
			appState.ErrorMessage = fmt.Sprintf("Error al consultar factores: %v", err)
			w.Invalidate()
			return
		}
		if len(factores) == 0 {
			appState.ErrorMessage = "No se encontraron factores"
			w.Invalidate()
			return
		}
		appState.Emision = factores[0]["Emision"].(time.Time).Format("2006-01-02")
		log.Println("Emision:", appState.Emision)
		appState.Facturas, err = db.GetFacturasParaFacturacion(appState.Emision)
		if err != nil {
			appState.ErrorMessage = fmt.Sprintf("Error al consultar lecturas pendientes: %v", err)
			w.Invalidate()
			return
		}

		if len(appState.Facturas) > 0 {
			appState.ErrorMessage = "Hay lecturas pendientes"
			w.Invalidate()
			return
		}

		// Change the view to the main screen
		appState.CurrentView = "main"
		w.Invalidate()
	}()

	for {
		// listen for events in the window.
		switch e := w.Event().(type) {

		// this is sent when the application should re-render.
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			layout.Flex{
				Axis:    layout.Vertical,
				Spacing: layout.SpaceEnd,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return material.H4(th, "Emisión "+appState.Emision).Layout(gtx)
					})
				}),
				layout.Rigid(
					func(gtx C) D {
						return layout.Center.Layout(gtx, func(gtx C) D {
							return material.H6(th, "Verificando facturas pendientes...").Layout(gtx)
						})
					},
				),
				layout.Rigid(
					func(gtx C) D {
						return layout.Center.Layout(gtx, func(gtx C) D {
							return material.Loader(th).Layout(gtx)
						})
					},
				),
				layout.Rigid(
					func(gtx C) D {
						if appState.ErrorMessage != "" {
							return layout.Center.Layout(gtx, func(gtx C) D {
								// change color to red
								th.Palette.Fg = color.NRGBA(colornames.Red500)
								return material.H2(th, appState.ErrorMessage).Layout(gtx)
							})
						}
						return D{}
					},
				),

				//mostrar lecturas pendiente abonado
				layout.Rigid(
					func(gtx C) D {
						if len(appState.Facturas) == 0 {
							return D{}
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(
								func(gtx C) D {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(material.Label(th, th.TextSize, "Lecturas pendientes").Layout),
										layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
										// create table widget with data
										layout.Rigid(func(gtx C) D {
											return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
												// create table header
												layout.Rigid(func(gtx C) D {
													return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
														layout.Rigid(material.Label(th, th.TextSize, "Abonado").Layout),
														layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
														layout.Rigid(material.Label(th, th.TextSize, "Lectura").Layout),
													)
												}),
												// create table rows
												layout.Rigid(func(gtx C) D {
													var rows []layout.FlexChild
													for _, factura := range appState.Facturas {
														rows = append(rows, layout.Rigid(func(gtx C) D {
															return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
																layout.Rigid(material.Label(th, th.TextSize, factura.Abonado).Layout),
																layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
																layout.Rigid(material.Label(th, th.TextSize, fmt.Sprintf("%.2f", factura.ConM3)).Layout),
															)
														}))
													}
													return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
												}),
											)
										}),
									)
								},
							),
						)
					},
				),
			)

			e.Frame(gtx.Ops)

		// this is sent when the application is closed.
		case app.DestroyEvent:
			return e.Err
		}

		// Check if the view should change to the main screen
		if appState.CurrentView == "main" {
			// Close the loading screen and open the main window
			w.Perform(system.ActionClose)

			go func() {
				// create new window
				w := new(app.Window)
				w.Option(app.Title("Facturación Masiva"))
				w.Option(app.Size(unit.Dp(500), unit.Dp(300)))
				if err := drawMainScreen(w, appState); err != nil {
					log.Fatal(err)
				}
				os.Exit(0)
			}()

			return nil
		}
	}
}

func drawMainScreen(w *app.Window, appState *AppState) error {
	// ops are the operations from the UI
	var ops op.Ops

	// startButton is a clickable widget
	var startButton widget.Clickable

	// is the process running?
	var running bool

	// th defines the material design style
	th := material.NewTheme()

	// Steps and their status
	steps := []Step{
		{"Consultar y verificar", Pending, false},
		{"Facturacion", Pending, false},
		{"Revision y validacion", Pending, false},
	}

	// Total progress
	var totalProgress float32

	// Error dialog
	var errorDialog widget.Clickable
	var errorMessage string

	// Progress info label
	var progressInfoText string

	// listen for events in the incrementor channel
	go func() {
		for range progressIncrementer {
			if running {
				// Force a redraw by invalidating the frame
				w.Invalidate()
			}
		}
	}()

	for {
		// listen for events in the window.
		switch e := w.Event().(type) {

		// this is sent when the application should re-render.
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Let's try out the flexbox layout concept
			if startButton.Clicked(gtx) {
				// Start (or stop) the process
				if running {
					return nil
				}
				running = true

				defer func() {
					running = false
					w.Invalidate()
				}()

				// Resetting the progress
				if totalProgress >= 1 {
					totalProgress = 0
					for i := range steps {
						steps[i].status = Pending
						steps[i].hasError = false
					}
				}
				go func() {
					// Step 1: Consultar a la base de datos
					steps[0].status = Processing
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
					log.Println("Emision:", emision)
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

					totalFacturas := len(facturas)

					// selecionar la primera factura
					// facturas = facturas[0:100]

					fmt.Printf("total %d, seleccionadas %d", totalFacturas, len(facturas))

					steps[0].status = Completed

					steps[1].status = Processing
					procesados, exitos, fallos := facturacionMasiva(facturas, &totalProgress, w, &progressInfoText)
					steps[1].status = Completed

					// Step 4: Verificar integridad de datos
					// Simulate data integrity check
					steps[2].status = Processing
					time.Sleep(1 * time.Second)
					steps[2].status = Completed

					progressInfoText = fmt.Sprintf("Procesando factura %d/%d, exitoso = %d, errores = %d", len(procesados), len(facturas), len(exitos), len(fallos))
					running = false
					w.Invalidate()
				}()
			}

			if errorDialog.Clicked(gtx) {
				errorMessage = ""
				w.Invalidate()
			}

			layout.Flex{
				// Vertical alignment, from top to bottom
				Axis: layout.Vertical,
				// Empty space is left at the start, i.e. at the top
				Spacing: layout.SpaceStart,
			}.Layout(
				gtx,

				// Steps and their status
				layout.Rigid(
					func(gtx C) D {
						var flexChildren []layout.FlexChild
						for i := range steps {
							step := steps[i]
							flexChildren = append(flexChildren, layout.Rigid(
								func(gtx C) D {
									return layout.Inset{
										Top:    unit.Dp(5),
										Bottom: unit.Dp(5),
										Left:   unit.Dp(10),
										Right:  unit.Dp(10),
									}.Layout(gtx, func(gtx C) D {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx C) D {
												var icon *widget.Icon
												var _widget layout.Widget
												switch step.status {
												case Pending:
													icon, _ = widget.NewIcon(icons.ToggleCheckBoxOutlineBlank)
													_widget = func(gtx C) D {
														return icon.Layout(gtx, color.NRGBA(colornames.Amber400))
													}
												case Processing:
													_widget = func(gtx C) D {
														// gtx.Constraints = layout.Exact(image.Pt(16, 16))
														return material.Loader(th).Layout(gtx)
													}
												case Completed:
													icon, _ = widget.NewIcon(icons.ToggleCheckBox)
													_widget = func(gtx C) D {
														return icon.Layout(gtx, color.NRGBA(colornames.Amber400))
													}
												}
												if step.hasError {
													return icon.Layout(gtx, color.NRGBA(colornames.Red500))
												}
												return _widget(gtx)
											}),
											layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
											layout.Rigid(material.Label(th, th.TextSize, step.name).Layout),
										)
									})
								},
							))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, flexChildren...)
					},
				),

				// Progress info label
				layout.Rigid(
					func(gtx C) D {
						return layout.Inset{
							Top:    unit.Dp(5),
							Bottom: unit.Dp(5),
							Left:   unit.Dp(10),
							Right:  unit.Dp(10),
						}.Layout(gtx, func(gtx C) D {
							return material.Label(th, th.TextSize, progressInfoText).Layout(gtx)
						})
					},
				),

				// The progressbar
				layout.Rigid(
					func(gtx C) D {
						bar := material.ProgressBar(th, totalProgress)
						bar.Height = unit.Dp(15)
						return layout.Inset{
							Top:    unit.Dp(5),
							Bottom: unit.Dp(5),
							Left:   unit.Dp(10),
							Right:  unit.Dp(10),
						}.Layout(gtx, bar.Layout)
					},
				),

				// The button
				layout.Rigid(
					func(gtx C) D {
						// We start by defining a set of margins
						margins := layout.Inset{
							Top:    unit.Dp(25),
							Bottom: unit.Dp(25),
							Right:  unit.Dp(35),
							Left:   unit.Dp(35),
						}
						// Then we lay out within those margins
						return margins.Layout(gtx,
							func(gtx C) D {
								// The text on the button depends on program state
								var text string = "Iniciar"

								if running && totalProgress < 1 {
									text = "Pausar"
								}

								// if running && totalProgress >= 1 {
								// 	text = "Finalizar"
								// }
								// if !running {
								// 	text = errorMessage
								// }

								newbutton := material.Button(th, &startButton, text)
								return newbutton.Layout(gtx)
							},
						)
					},
				),

				// Error dialog
				layout.Rigid(
					func(gtx C) D {
						if errorMessage != "" {
							return layout.Center.Layout(gtx, func(gtx C) D {
								return material.H2(th, errorMessage).Layout(gtx)
							})
						}
						return D{}
					},
				),
			)
			e.Frame(gtx.Ops)

		// this is sent when the application is closed.
		case app.DestroyEvent:
			return e.Err
		}
	}
}


func facturacionMasiva(facturas []db.Factura, totalProgress *float32, w *app.Window, progressInfoText *string) ([]db.Factura, []db.Factura, []db.Factura) {
    fe := api.NewFacturacionElectronica(api.ApiConfig{
        Url:    "http://192.168.0.102:3001",
        ApiKey: "8b6d1b35ea7998191033237d588abd859e24af22895e2ec7574c8748a3be2cdcf5153f9bfca1d617167c6128ecd2880e7225fa0c7ada6461ca55fc52daec0fe4e1acee0d380323fccdb67b8a1cbf40c4d2718988e5bf5f7d95f98733af5152b84f0ceb500359fe385916bc775323a2d154ff0acc694e4ce36d8b696eea07c1498e5c642440022eef8a954eeee90dfd8d0c1d5d935cc5a768e640dd1fc764726fb5f7fca2c8ccb238d381fe03c8cb89a75f61e3fe32e7a984a7e8470b795a4df3637edcd913bdce45304a62ed8bd8485147ce0bd29dcbd8a82276568497146a02f6536288c3bb1f01c5c328ad92fff30568a1781634f8ed6052340d082d04dd81b20ed6ea77a4cecf1ab66d96b6a97107",
    })
    const maxGoroutines = 100
    sem := make(chan struct{}, maxGoroutines)
    var wg sync.WaitGroup

    procesados := []db.Factura{}
    exitos := []db.Factura{}
    fallos := []db.Factura{}

    for _, factura := range facturas {
        sem <- struct{}{}
        wg.Add(1)
        go func(factura db.Factura) {
            defer func() {
                <-sem
                wg.Done()
            }()
            
            result, err := fe.FacturaServicios(
                time.Now(),
                factura.ConM3, factura.ImpTotal, factura.ImpAlcanta,
                factura.ImpRep, factura.ImpFactura, factura.ImpRecargo,
                factura.ImpLey1886, factura.Razon, factura.Abonado,
                factura.Nit, factura.Zona, factura.Calle, factura.NumFactura,
            )
            procesados = append(procesados, factura)
            if err != nil {
                log.Println("Error generating factura:", err)
                fallos = append(fallos, factura)
                return
            }

            cuf := result["cuf"].(string)
            err = db.UpdateFacturaCodigoControl(factura.FacturaID, cuf)
            if err != nil {
                log.Println("Error updating factura codigo control:", err)
                fallos = append(fallos, factura)
                return
            }

            *totalProgress += 1.0 / float32(len(facturas))
            if *totalProgress >= 1 {
                *totalProgress = 1
            }
            exitos = append(exitos, factura)
            *progressInfoText = fmt.Sprintf("Procesando factura %d/%d, exitoso = %d, errores = %d", len(procesados), len(facturas), len(exitos), len(fallos))
            w.Invalidate()
        }(factura)
    }

    wg.Wait()
    return procesados, exitos, fallos
}