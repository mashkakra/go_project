package main

import (
	"errors"
	"testing"
	"unicode/utf8"
)

func Testt(t *testing.T) {
	var cases = []struct {
		in  []byte
		exp int
		err error
	}{{
		[]byte("異體字"),
		3,
		nil,
	}, {
		[]byte(""),
		0,
		nil,
	}, {[]byte("12*ёё"),
		0,
		errors.New("invalid utf8"),
	},
	}
	for _, test := range cases {
		test := test
		got, err := GetUTFLength(test.in)
		if got != test.exp && err != test.err {
			t.Errorf("r")
		}
	}
}

var ErrInvalidUTF8 = errors.New("invalid utf8")

func GetUTFLength(input []byte) (int, error) {
	if !utf8.Valid(input) {
		return 0, ErrInvalidUTF8
	}

	return utf8.RuneCount(input), nil
}
