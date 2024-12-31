package ui

import (
	"fmt"
	"image/color"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/colornames"
)

type LoadingView struct {
	AppState *AppState
}

func (v *LoadingView) Layout(gtx layout.Context) layout.Dimensions {
	th := material.NewTheme()

	layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceEnd,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				return material.H4(th, "Emisión "+v.AppState.Emision).Layout(gtx)
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
				if v.AppState.ErrorMessage != "" {
					return layout.Center.Layout(gtx, func(gtx C) D {
						// change color to red
						th.Palette.Fg = color.NRGBA(colornames.Red500)
						return material.H2(th, v.AppState.ErrorMessage).Layout(gtx)
					})
				}
				return D{}
			},
		),

		//mostrar lecturas pendiente abonado
		layout.Rigid(
			func(gtx C) D {
				if len(v.AppState.Facturas) == 0 {
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
											for _, factura := range v.AppState.Facturas {
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

	return layout.Dimensions{}
}
