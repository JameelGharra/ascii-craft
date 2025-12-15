package utils

import "log"

func Assert(truth bool, message string) {
	if !truth {
		log.Fatal(message)
	}
}
