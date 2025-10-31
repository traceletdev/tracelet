package rules

import "fmt"

func floatToString(f float64, prec int) string {
    format := fmt.Sprintf("%%.%df", prec)
    return fmt.Sprintf(format, f)
}


