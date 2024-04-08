// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fly

func RegionName(code string) string {
	if name, ok := regions[code]; ok {
		return name
	}
	return code
}

var regions = map[string]string{
	"ams": "Amsterdam, Netherlands",
	"cdg": "Paris, France",
	"den": "Denver, Colorado (US)",
	"dfw": "Dallas, Texas (US)",
	"ewr": "Secaucus, NJ (US)",
	"fra": "Frankfurt, Germany",
	"gru": "São Paulo",
	"hkg": "Hong Kong, Hong Kong",
	"iad": "Ashburn, Virginia (US)",
	"jnb": "Johannesburg, South Africa",
	"lax": "Los Angeles, California (US)",
	"lhr": "London, United Kingdom",
	"maa": "Chennai (Madras), India",
	"mad": "Madrid, Spain",
	"mia": "Miami, Florida (US)",
	"nrt": "Tokyo, Japan",
	"ord": "Chicago, Illinois (US)",
	"otp": "Bucharest, Romania",
	"scl": "Santiago, Chile",
	"sea": "Seattle, Washington (US)",
	"sin": "Singapore",
	"sjc": "Sunnyvale, California (US)",
	"syd": "Sydney, Australia",
	"waw": "Warsaw, Poland",
	"yul": "Montreal, Canada",
	"yyz": "Toronto, Canada",
}
