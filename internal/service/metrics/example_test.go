// SPDX-License-Identifier: AGPL-3.0-or-later

package metrics

import "fmt"

// The worked examples in the help center are these outputs.

func ExampleWilson() {
	iv := Wilson(5, 20)
	fmt.Printf("5 of 20: %.1f%% to %.1f%%\n", iv.Lo*100, iv.Hi*100)
	iv = Wilson(4, 6)
	fmt.Printf("4 of 6: %.1f%% to %.1f%%\n", iv.Lo*100, iv.Hi*100)
	// Output:
	// 5 of 20: 11.2% to 46.9%
	// 4 of 6: 30.0% to 90.3%
}

func ExampleChange() {
	fmt.Println(Change(18, 30, 6, 30))
	fmt.Println(Change(6, 20, 5, 20))
	iv := NewcombeDiff(6, 30, 18, 30)
	fmt.Printf("difference %.1f to %.1f points\n", iv.Lo*100, iv.Hi*100)
	// Output:
	// down
	// flat
	// difference -58.6 to -15.3 points
}
