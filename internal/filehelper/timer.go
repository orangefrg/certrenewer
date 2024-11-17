package filehelper

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Duration struct {
	Duration time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	duration, err := StringToTime(s)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}

func StringToTime(input string) (time.Duration, error) {
	var totalDuration time.Duration
	var parts []string
	current := ""

	isUnit := false

	for _, char := range input {
		if (char >= '0' && char <= '9') || char == '.' {
			if isUnit {
				parts = append(parts, current)
				current = ""
				isUnit = false
			}
			current += string(char)
		} else {
			isUnit = true
			current += string(char)
		}
	}
	if len(current) > 0 {
		parts = append(parts, current)
	}
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		pos := -1
		for i, p := range part {
			if p < '0' || p > '9' {
				if p != '.' {
					pos = i
					break
				}
			}
		}

		if pos == -1 {
			return 0, errors.New("invalid time format")
		}

		numStr := part[:pos]
		unitStr := part[pos:]

		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("could not parse number: %w", err)
		}

		unitStr = strings.ToLower(unitStr)

		switch unitStr {
		case "d":
			totalDuration += time.Duration(num * float64(24*time.Hour))
		case "h":
			totalDuration += time.Duration(num * float64(time.Hour))
		case "m":
			totalDuration += time.Duration(num * float64(time.Minute))
		case "s":
			totalDuration += time.Duration(num * float64(time.Second))
		case "ms":
			totalDuration += time.Duration(num * float64(time.Millisecond))
		case "us", "µs":
			totalDuration += time.Duration(num * float64(time.Microsecond))
		case "ns":
			totalDuration += time.Duration(num * float64(time.Nanosecond))
		default:
			return 0, fmt.Errorf("unsupported time unit '%s'", unitStr)
		}
	}

	return totalDuration, nil
}
