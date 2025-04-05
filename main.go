package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"github.com/urfave/cli/v2"
)

// Constantes financieras para México
const (
	ISR              = 0.20  // Impuesto Sobre la Renta para intereses (20%)
	INFLACION_ANUAL  = 0.042 // Inflación anual estimada (4.2%)
	PAGO_MINIMO      = 0.05  // Porcentaje de pago mínimo típico (5%)
	ARCHIVO_TARJETAS = "tarjetas.json"
)

// TarjetaDebito representa la información de una tarjeta de débito
type TarjetaDebito struct {
	Nombre            string  `json:"nombre"`
	Banco             string  `json:"banco"`
	TasaRendimiento   float64 `json:"tasa_rendimiento"` // Tasa anual
	SaldoMinimo       float64 `json:"saldo_minimo"`
	ComisionAnual     float64 `json:"comision_anual"`
	ComisionInactividad float64 `json:"comision_inactividad"`
}

// TarjetaCredito representa la información de una tarjeta de crédito
type TarjetaCredito struct {
	Nombre           string  `json:"nombre"`
	Banco            string  `json:"banco"`
	TasaInteres      float64 `json:"tasa_interes"` // Tasa anual
	CAT              float64 `json:"cat"`          // Costo Anual Total
	ComisionAnual    float64 `json:"comision_anual"`
	LimiteCredito    float64 `json:"limite_credito"`
	BeneficiosCashback float64 `json:"beneficios_cashback"` // Porcentaje de cashback
	MesesSinIntereses bool    `json:"meses_sin_intereses"`  // Ofrece MSI
}

// Tarjetas almacena todas las tarjetas guardadas
type Tarjetas struct {
	Debito  []TarjetaDebito  `json:"debito"`
	Credito []TarjetaCredito `json:"credito"`
}

// CargarTarjetas carga las tarjetas desde el archivo JSON
func CargarTarjetas() (Tarjetas, error) {
	var tarjetas Tarjetas

	// Verifica si el archivo existe
	if _, err := os.Stat(ARCHIVO_TARJETAS); os.IsNotExist(err) {
		// Si no existe, crea un archivo con estructura vacía
		tarjetas = Tarjetas{
			Debito:  []TarjetaDebito{},
			Credito: []TarjetaCredito{},
		}
		
		data, err := json.MarshalIndent(tarjetas, "", "  ")
		if err != nil {
			return tarjetas, err
		}
		
		err = ioutil.WriteFile(ARCHIVO_TARJETAS, data, 0644)
		return tarjetas, err
	}

	// Lee el archivo existente
	data, err := ioutil.ReadFile(ARCHIVO_TARJETAS)
	if err != nil {
		return tarjetas, err
	}

	err = json.Unmarshal(data, &tarjetas)
	return tarjetas, err
}

// GuardarTarjetas guarda las tarjetas en el archivo JSON
func GuardarTarjetas(tarjetas Tarjetas) error {
	data, err := json.MarshalIndent(tarjetas, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(ARCHIVO_TARJETAS, data, 0644)
}

// CalcularRendimientoReal calcula el rendimiento real después de impuestos e inflación
func CalcularRendimientoReal(tarjeta TarjetaDebito, saldo float64) (float64, float64, float64) {
	// Calculamos solo si el saldo es mayor al mínimo requerido
	if saldo < tarjeta.SaldoMinimo {
		return 0, 0, saldo - tarjeta.ComisionAnual
	}
	
	// Rendimiento anual bruto
	rendimientoBruto := saldo * tarjeta.TasaRendimiento
	
	// Impuesto sobre rendimiento
	impuestos := rendimientoBruto * ISR
	
	// Rendimiento neto después de impuestos
	rendimientoNeto := rendimientoBruto - impuestos
	
	// Pérdida por inflación
	perdidaInflacion := saldo * INFLACION_ANUAL
	
	// Rendimiento real (considerando inflación)
	rendimientoReal := rendimientoNeto - perdidaInflacion - tarjeta.ComisionAnual
	
	// Saldo final después de un año
	saldoFinal := saldo + rendimientoReal
	
	return rendimientoReal, rendimientoReal / saldo * 100, saldoFinal
}

// CalcularCostoCredito calcula el costo total de usar la tarjeta de crédito
func CalcularCostoCredito(tarjeta TarjetaCredito, deuda float64, pagoMensual float64) (float64, int, float64) {
	// Si el pago mensual es menor al pago mínimo, ajustamos
	pagoMinimoMensual := deuda * PAGO_MINIMO
	if pagoMensual < pagoMinimoMensual {
		pagoMensual = pagoMinimoMensual
	}
	
	// Calculamos la tasa de interés mensual
	tasaMensual := tarjeta.TasaInteres / 12
	
	// Variables para seguimiento
	deudaActual := deuda
	meses := 0
	interesTotal := 0.0
	
	// Simulamos los pagos mensuales hasta liquidar la deuda
	for deudaActual > 0 && meses < 1000 { // Límite para evitar bucle infinito
		// Interés del mes
		interesMes := deudaActual * tasaMensual
		interesTotal += interesMes
		
		// Aplicamos el pago mensual
		pago := math.Min(pagoMensual, deudaActual + interesMes)
		deudaActual = deudaActual + interesMes - pago
		
		meses++
		
		// Si la deuda es muy pequeña, la consideramos pagada
		if deudaActual < 0.01 {
			deudaActual = 0
		}
	}
	
	// Costo total = intereses + comisión anual (prorrateada por los meses)
	comisionPeriodo := tarjeta.ComisionAnual * float64(meses) / 12
	costoTotal := interesTotal + comisionPeriodo
	
	// Calculamos el beneficio de cashback (si aplica)
	beneficioCashback := deuda * tarjeta.BeneficiosCashback
	
	// Costo neto después de beneficios
	costoNeto := costoTotal - beneficioCashback
	
	return costoNeto, meses, costoNeto / deuda * 100
}

func main() {
	app := &cli.App{
		Name:  "finmex",
		Usage: "Calculadora financiera para productos financieros mexicanos",
		Commands: []*cli.Command{
			{
				Name:  "debito",
				Usage: "Operaciones con tarjetas de débito",
				Subcommands: []*cli.Command{
					{
						Name:  "agregar",
						Usage: "Agregar una nueva tarjeta de débito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							var tarjeta TarjetaDebito
							
							fmt.Print("Nombre de la tarjeta: ")
							fmt.Scan(&tarjeta.Nombre)
							
							fmt.Print("Banco emisor: ")
							fmt.Scan(&tarjeta.Banco)
							
							fmt.Print("Tasa de rendimiento anual (decimal, ej: 0.05 para 5%): ")
							fmt.Scan(&tarjeta.TasaRendimiento)
							
							fmt.Print("Saldo mínimo requerido: ")
							fmt.Scan(&tarjeta.SaldoMinimo)
							
							fmt.Print("Comisión anual: ")
							fmt.Scan(&tarjeta.ComisionAnual)
							
							fmt.Print("Comisión por inactividad (mensual): ")
							fmt.Scan(&tarjeta.ComisionInactividad)
							
							tarjetas.Debito = append(tarjetas.Debito, tarjeta)
							
							err = GuardarTarjetas(tarjetas)
							if err != nil {
								return fmt.Errorf("Error al guardar tarjeta: %v", err)
							}
							
							fmt.Printf("Tarjeta de débito '%s' agregada exitosamente\n", tarjeta.Nombre)
							return nil
						},
					},
					{
						Name:  "analizar",
						Usage: "Analizar rendimiento de una tarjeta de débito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Debito) == 0 {
								return fmt.Errorf("No hay tarjetas de débito registradas")
							}
							
							fmt.Println("Tarjetas de débito disponibles:")
							for i, t := range tarjetas.Debito {
								fmt.Printf("%d. %s (%s)\n", i+1, t.Nombre, t.Banco)
							}
							
							var seleccion int
							fmt.Print("Selecciona una tarjeta (número): ")
							fmt.Scan(&seleccion)
							
							if seleccion < 1 || seleccion > len(tarjetas.Debito) {
								return fmt.Errorf("Selección inválida")
							}
							
							tarjeta := tarjetas.Debito[seleccion-1]
							
							var saldo float64
							fmt.Print("Ingresa el saldo promedio a mantener: ")
							fmt.Scan(&saldo)
							
							rendimiento, rendimientoPct, saldoFinal := CalcularRendimientoReal(tarjeta, saldo)
							
							fmt.Println("\n=== Análisis de Rendimiento ===")
							fmt.Printf("Tarjeta: %s (%s)\n", tarjeta.Nombre, tarjeta.Banco)
							fmt.Printf("Tasa nominal: %.2f%%\n", tarjeta.TasaRendimiento*100)
							fmt.Printf("Saldo inicial: $%.2f\n", saldo)
							fmt.Printf("Rendimiento bruto anual: $%.2f\n", saldo*tarjeta.TasaRendimiento)
							fmt.Printf("Impuestos (ISR %.0f%%): $%.2f\n", ISR*100, saldo*tarjeta.TasaRendimiento*ISR)
							fmt.Printf("Pérdida por inflación (%.1f%%): $%.2f\n", INFLACION_ANUAL*100, saldo*INFLACION_ANUAL)
							fmt.Printf("Comisión anual: $%.2f\n", tarjeta.ComisionAnual)
							fmt.Printf("Rendimiento real anual: $%.2f (%.2f%%)\n", rendimiento, rendimientoPct)
							
							if rendimiento > 0 {
								fmt.Printf("RESULTADO: Tu dinero GANA valor real ($%.2f después de un año)\n", saldoFinal)
							} else {
								fmt.Printf("RESULTADO: Tu dinero PIERDE valor real ($%.2f después de un año)\n", saldoFinal)
							}
							
							return nil
						},
					},
					{
						Name:  "listar",
						Usage: "Listar tarjetas de débito registradas",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Debito) == 0 {
								fmt.Println("No hay tarjetas de débito registradas")
								return nil
							}
							
							w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
							fmt.Fprintln(w, "Nombre\tBanco\tRendimiento\tSaldo Mínimo\tComisión Anual")
							fmt.Fprintln(w, "------\t-----\t-----------\t------------\t--------------")
							
							for _, t := range tarjetas.Debito {
								fmt.Fprintf(w, "%s\t%s\t%.2f%%\t$%.2f\t$%.2f\n",
									t.Nombre, t.Banco, t.TasaRendimiento*100, 
									t.SaldoMinimo, t.ComisionAnual)
							}
							
							w.Flush()
							return nil
						},
					},
				},
			},
			{
				Name:  "credito",
				Usage: "Operaciones con tarjetas de crédito",
				Subcommands: []*cli.Command{
					{
						Name:  "agregar",
						Usage: "Agregar una nueva tarjeta de crédito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							var tarjeta TarjetaCredito
							
							fmt.Print("Nombre de la tarjeta: ")
							fmt.Scan(&tarjeta.Nombre)
							
							fmt.Print("Banco emisor: ")
							fmt.Scan(&tarjeta.Banco)
							
							fmt.Print("Tasa de interés anual (decimal, ej: 0.36 para 36%): ")
							fmt.Scan(&tarjeta.TasaInteres)
							
							fmt.Print("CAT (decimal, ej: 0.45 para 45%): ")
							fmt.Scan(&tarjeta.CAT)
							
							fmt.Print("Comisión anual: ")
							fmt.Scan(&tarjeta.ComisionAnual)
							
							fmt.Print("Límite de crédito: ")
							fmt.Scan(&tarjeta.LimiteCredito)
							
							fmt.Print("Porcentaje de cashback (decimal, ej: 0.02 para 2%): ")
							fmt.Scan(&tarjeta.BeneficiosCashback)
							
							var msiStr string
							fmt.Print("¿Ofrece meses sin intereses? (s/n): ")
							fmt.Scan(&msiStr)
							tarjeta.MesesSinIntereses = strings.ToLower(msiStr) == "s"
							
							tarjetas.Credito = append(tarjetas.Credito, tarjeta)
							
							err = GuardarTarjetas(tarjetas)
							if err != nil {
								return fmt.Errorf("Error al guardar tarjeta: %v", err)
							}
							
							fmt.Printf("Tarjeta de crédito '%s' agregada exitosamente\n", tarjeta.Nombre)
							return nil
						},
					},
					{
						Name:  "analizar",
						Usage: "Analizar costo de una tarjeta de crédito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Credito) == 0 {
								return fmt.Errorf("No hay tarjetas de crédito registradas")
							}
							
							fmt.Println("Tarjetas de crédito disponibles:")
							for i, t := range tarjetas.Credito {
								fmt.Printf("%d. %s (%s)\n", i+1, t.Nombre, t.Banco)
							}
							
							var seleccion int
							fmt.Print("Selecciona una tarjeta (número): ")
							fmt.Scan(&seleccion)
							
							if seleccion < 1 || seleccion > len(tarjetas.Credito) {
								return fmt.Errorf("Selección inválida")
							}
							
							tarjeta := tarjetas.Credito[seleccion-1]
							
							var deuda float64
							fmt.Print("Ingresa el monto de la deuda/compra: ")
							fmt.Scan(&deuda)
							
							var pagoMensual float64
							fmt.Print("Ingresa el pago mensual que planeas hacer: ")
							fmt.Scan(&pagoMensual)
							
							pagoMinimo := deuda * PAGO_MINIMO
							if pagoMensual < pagoMinimo {
								fmt.Printf("AVISO: El pago ingresado es menor al pago mínimo. Se ajustará a $%.2f\n", pagoMinimo)
								pagoMensual = pagoMinimo
							}
							
							costo, meses, costoPct := CalcularCostoCredito(tarjeta, deuda, pagoMensual)
							
							fmt.Println("\n=== Análisis de Crédito ===")
							fmt.Printf("Tarjeta: %s (%s)\n", tarjeta.Nombre, tarjeta.Banco)
							fmt.Printf("Deuda/Compra: $%.2f\n", deuda)
							fmt.Printf("Tasa de interés anual: %.2f%%\n", tarjeta.TasaInteres*100)
							fmt.Printf("CAT: %.2f%%\n", tarjeta.CAT*100)
							fmt.Printf("Pago mensual: $%.2f\n", pagoMensual)
							fmt.Printf("Tiempo para liquidar: %d meses (%.1f años)\n", meses, float64(meses)/12)
							
							if tarjeta.BeneficiosCashback > 0 {
								fmt.Printf("Beneficio por cashback (%.1f%%): $%.2f\n", 
									tarjeta.BeneficiosCashback*100, deuda*tarjeta.BeneficiosCashback)
							}
							
							fmt.Printf("Costo total del crédito: $%.2f (%.2f%% del monto original)\n", costo, costoPct)
							fmt.Printf("Monto total pagado: $%.2f\n", deuda+costo)
							
							return nil
						},
					},
					{
						Name:  "listar",
						Usage: "Listar tarjetas de crédito registradas",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Credito) == 0 {
								fmt.Println("No hay tarjetas de crédito registradas")
								return nil
							}
							
							w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
							fmt.Fprintln(w, "Nombre\tBanco\tInterés\tCAT\tComisión Anual\tLímite\tCashback\tMSI")
							fmt.Fprintln(w, "------\t-----\t-------\t---\t--------------\t------\t--------\t---")
							
							for _, t := range tarjetas.Credito {
								msi := "No"
								if t.MesesSinIntereses {
									msi = "Sí"
								}
								
								fmt.Fprintf(w, "%s\t%s\t%.2f%%\t%.2f%%\t$%.2f\t$%.2f\t%.2f%%\t%s\n",
									t.Nombre, t.Banco, t.TasaInteres*100, t.CAT*100,
									t.ComisionAnual, t.LimiteCredito, t.BeneficiosCashback*100, msi)
							}
							
							w.Flush()
							return nil
						},
					},
				},
			},
			{
				Name:  "comparar",
				Usage: "Comparar tarjetas registradas",
				Subcommands: []*cli.Command{
					{
						Name:  "debito",
						Usage: "Comparar tarjetas de débito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Debito) < 2 {
								return fmt.Errorf("Se necesitan al menos 2 tarjetas de débito para comparar")
							}
							
							var saldo float64
							fmt.Print("Ingresa el saldo promedio a mantener para la comparación: ")
							fmt.Scan(&saldo)
							
							fmt.Println("\n=== Comparación de Tarjetas de Débito ===")
							fmt.Printf("Saldo a comparar: $%.2f\n\n", saldo)
							
							w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
							fmt.Fprintln(w, "Nombre\tBanco\tRend. Nominal\tRend. Real\tSaldo Final\tResultado")
							fmt.Fprintln(w, "------\t-----\t------------\t---------\t-----------\t--------")
							
							for _, t := range tarjetas.Debito {
								rendimiento, rendimientoPct, saldoFinal := CalcularRendimientoReal(t, saldo)
								
								resultado := "PIERDE"
								if rendimiento > 0 {
									resultado = "GANA"
								}
								
								fmt.Fprintf(w, "%s\t%s\t%.2f%%\t%.2f%%\t$%.2f\t%s\n",
									t.Nombre, t.Banco, t.TasaRendimiento*100, rendimientoPct,
									saldoFinal, resultado)
							}
							
							w.Flush()
							return nil
						},
					},
					{
						Name:  "credito",
						Usage: "Comparar tarjetas de crédito",
						Action: func(c *cli.Context) error {
							tarjetas, err := CargarTarjetas()
							if err != nil {
								return fmt.Errorf("Error al cargar tarjetas: %v", err)
							}
							
							if len(tarjetas.Credito) < 2 {
								return fmt.Errorf("Se necesitan al menos 2 tarjetas de crédito para comparar")
							}
							
							var deuda float64
							fmt.Print("Ingresa el monto de la deuda/compra para la comparación: ")
							fmt.Scan(&deuda)
							
							var pagoMensual float64
							fmt.Print("Ingresa el pago mensual que planeas hacer: ")
							fmt.Scan(&pagoMensual)
							
							fmt.Println("\n=== Comparación de Tarjetas de Crédito ===")
							fmt.Printf("Deuda a comparar: $%.2f\n", deuda)
							fmt.Printf("Pago mensual: $%.2f\n\n", pagoMensual)
							
							w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
							fmt.Fprintln(w, "Nombre\tBanco\tCAT\tCosto Total\tMeses\tCashback\tMSI")
							fmt.Fprintln(w, "------\t-----\t---\t-----------\t-----\t--------\t---")
							
							for _, t := range tarjetas.Credito {
								costo, meses, _ := CalcularCostoCredito(t, deuda, pagoMensual)
								
								msi := "No"
								if t.MesesSinIntereses {
									msi = "Sí"
								}
								
								fmt.Fprintf(w, "%s\t%s\t%.2f%%\t$%.2f\t%d\t%.2f%%\t%s\n",
									t.Nombre, t.Banco, t.CAT*100, costo, meses,
									t.BeneficiosCashback*100, msi)
							}
							
							w.Flush()
							return nil
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

