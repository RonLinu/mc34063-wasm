package main

import (
	"fmt"
	"math"
	"mc34063/swal"
	"strconv"
	"syscall/js"
)

const RESOURCES = "resources/"
const RIPPLE = 0.1

type UserValues struct {
	vin, vout, iout, freq, res1 float64
}

type Results struct {
	lmin, ct, cout, rsc, r2, rb float64
}

var fieldsNames = [...]string{"vin", "vout", "iout", "freq", "res1"}

var document = js.Global().Get("document")

// -------------------------------------
func main() {
	html_fields_handlers()
	html_buttons_handlers()

	localStorage_restore_fields()
    about()    
	select {}
}

// ------------------------- HTML functions ---------------------------

func getValueByID(id string) string {
	return document.Call("getElementById", id).Get("value").String()
}

// -------------------------------------
func setInnerHTMLByID(id, html string) {
	document.Call("getElementById", id).Set("innerHTML", html)
}

// -------------------------------------
func html_fields_handlers() {
	// Define event handler function for a field input
	onInputChange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		calculateBtn := document.Call("getElementById", "calculate")
		calculateBtn.Set("disabled", false)
		clearFieldsErrors()
		return nil
	})

	// Get all input fields
	fields := document.Call("querySelectorAll", "form input")

	length := fields.Length()
	for i := 0; i < length; i++ {
		input := fields.Index(i)
		input.Call("addEventListener", "input", onInputChange)
	}
}

// -------------------------------------
func html_buttons_handlers() {
	// Event handler for 'Calculate parts' button
	calculateHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		calculate()
		return nil
	})

	calculateBtn := document.Call("getElementById", "calculate")
	calculateBtn.Call("addEventListener", "click", calculateHandler)

	// Event handler for 'Save fields' button
	saveHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		localStorage_save_fields()
		return nil
	})

	saveBtn := document.Call("getElementById", "save")
	saveBtn.Call("addEventListener", "click", saveHandler)
}

// -------------------------------------
func localStorage_save_fields() {
	for _, field := range fieldsNames {
		js.Global().Get("localStorage").Call("setItem", field, getValueByID(field))
	}
	swal.ShowAlert("", "info", "center", "Fields have been saved!")
}

// -------------------------------------
func localStorage_restore_fields() {
	for _, field := range fieldsNames {
		value := js.Global().Get("localStorage").Call("getItem", field).String()
		if value != "<null>" {
			document.Call("getElementById", field).Set("value", value)
		}
	}
}

// --------------------------------------------------------------------
func about() {
    title := "MC34063 calculator \u00A9"
    msg := `This application calculates the value of all the components required
	to build a switching regulator based on the MC34063 chip.
    <br><br>
	The following configurations are supported:<br>
	- Step Down (buck)<br>
	- Step Up (boost)<br>
	- Inverter<br>
    `
	swal.ShowAlert(title, "", "left", msg)
}

// --------------------------------------------------------------------
func get_user_values() UserValues {
	vinValue := getValueByID("vin")
	vin, _ := strconv.ParseFloat(vinValue, 64)

	voutValue := getValueByID("vout")
	vout, _ := strconv.ParseFloat(voutValue, 64)

	ioutValue := getValueByID("iout")
	iout, _ := strconv.ParseFloat(ioutValue, 64)

	freqValue := getValueByID("freq")
	freq, _ := strconv.ParseFloat(freqValue, 64)

	res1Value := getValueByID("res1")
	res1, _ := strconv.ParseFloat(res1Value, 64)

	values := UserValues{vin, vout, iout, freq, res1}

	return values
}

// -------------------------------------------------------------------
func clearFieldsErrors() {
	// Remove pink background (error markers) in all fields
	for _, field := range fieldsNames {
		field := document.Call("getElementById", field)
		field.Get("style").Set("backgroundColor", "White")
	}

	// Clear the rest of the web page
	setInnerHTMLByID("results", "")
	setInnerHTMLByID("regulator-name", "Regulator name")
	document.Call("getElementById", "schematic").Set("src", RESOURCES + "splash.png")
}

// -------------------------------------------------------------------
func calculate() {
	showFieldError := func(id string) {
		// Put pink background (error marker) in the field
		field := document.Call("getElementById", id)
		field.Get("style").Set("backgroundColor", "LightPink")
	}

	clearFieldsErrors()

	nums := get_user_values()

	// check if user values exceed range limits
	errCount := 0

	if nums.vin < 5 || nums.vin > 40 {
		errCount++
		showFieldError("vin")
	}

	if (nums.vout < -40 || nums.vout > 40) || (nums.vout > -3 && nums.vout < 3) {
		errCount++
		showFieldError("vout")
	}

	if nums.iout < 5 || nums.iout > 1000 {
		errCount++
		showFieldError("iout")
	}

	if nums.freq < 20 || nums.freq > 100 {
		errCount++
		showFieldError("freq")
	}

	if nums.res1 < 1 || nums.res1 > 50 {
		errCount++
		showFieldError("res1")
	}

	// Show number of errors in the result area
	if errCount > 0 {
		plural := "s are"
		if errCount < 2 {
			plural = " is"
		}
		msg := fmt.Sprintf("%d field%s out of limits", errCount, plural)
		setInnerHTMLByID("results", msg)
		return
	}

	calculateBtn := document.Call("getElementById", "calculate")
	calculateBtn.Set("disabled", true)

	if nums.vout < 0 {
		inverter(nums)
	} else if nums.vout < nums.vin {
		step_down(nums)
	} else {
		step_up(nums)
	}
}

// --------------------------------------------------------------------
func show_results(r Results, title, schematic string) {
	resultStr := "<pre>"
	resultStr += "<u>" + title + "</u>\n"
	resultStr += fmt.Sprintf("L   = %.0f uH (min)\n", r.lmin*1e6)

	resultStr += fmt.Sprintf("Ct  = %.0f pF\n", r.ct*1e12)
	resultStr += fmt.Sprintf("Co  = %.0f uF (min)\n", r.cout*1e6)
	resultStr += fmt.Sprintf("Rsc = %.1f Ω\n", r.rsc)
	resultStr += fmt.Sprintf("R2  = %.1f KΩ\n", r.r2)

	if r.rb != 0.0 {
		resultStr += fmt.Sprintf("Rb  = %.0f Ω\n", r.rb)
	}
	resultStr += "</pre>"

	setInnerHTMLByID("results", resultStr)
	setInnerHTMLByID("regulator-name", title)
	document.Call("getElementById", "schematic").Set("src", RESOURCES + schematic)
}

// --------------------------------------------------------------------
func step_down(nums UserValues) {
	ratio := (nums.vout + 0.8) / (nums.vin - 0.8 - nums.vout)
	tontoff := 1.0 / (nums.freq * 1e3)
	toff := tontoff / (ratio + 1)
	ton_max := tontoff - toff
	ipeak := nums.iout / 1e3 * 2.0

	lmin := (nums.vin - 1 - nums.vout) / ipeak * ton_max
	ct := ton_max * 4e-5
	cout := (ipeak * tontoff) / (8 * RIPPLE)
	rsc := 0.33 / ipeak
	r2 := (nums.vout - 1.25) / 1.25 * nums.res1 // R1 & R2 are in Kohms
	rb := 0.0

	results := Results{lmin, ct, cout, rsc, r2, rb}
	show_results(results, "Step-Down regulator", "step_down.png")
}

// --------------------------------------------------------------------
func inverter(nums UserValues) {
	ratio := (math.Abs(nums.vout) + 0.8) / (nums.vin - 0.8)
	tontoff := 1.0 / (nums.freq * 1e3)
	toff := tontoff / (ratio + 1)
	ton := tontoff - toff
	ipeak := 2 * nums.iout / 1e3

	lmin := (nums.vin - 0.8) / ipeak * ton
	ct := ton * 4e-5
	cout := (nums.iout / 1e3 * ton) / RIPPLE
	rsc := 0.33 / ipeak
	r2 := ((math.Abs(nums.vout) - 1.25) / 1.25) * nums.res1
	rb := 0.0

	results := Results{lmin, ct, cout, rsc, r2, rb}
	show_results(results, "Inverter regulator", "inverter.png")
}

// --------------------------------------------------------------------
func step_up(nums UserValues) {
	ratio := (nums.vout + 0.8 - nums.vin) / (nums.vin - 1)
	tontoff := 1.0 / (nums.freq * 1e3)
	toff := tontoff / (ratio + 1)
	ton_max := tontoff - toff
	ipeak := nums.iout / 1e3 * (ratio + 1) * 2.0
	ib := ipeak/20 + 5e-3

	lmin := (nums.vin - 1) / ipeak * ton_max
	ct := ton_max * 4e-5
	cout := (nums.iout / 1e3 * ton_max) / RIPPLE
	rsc := 0.33 / ipeak
	r2 := ((nums.vout - 1.25) / 1.25) * nums.res1
	rb := ((nums.vin - 1) - ipeak) * rsc / ib

	results := Results{lmin, ct, cout, rsc, r2, rb}
	show_results(results, "Step-Up regulator", "step_up.png")
}
