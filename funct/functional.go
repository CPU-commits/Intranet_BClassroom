package funct

import (
	"reflect"
)

func Flat[T any](slide interface{}) []T {
	var result []T

	vSlide := reflect.ValueOf(slide)
	if vSlide.Kind() == reflect.Array || vSlide.Kind() == reflect.Slice {
		for i := 0; i < vSlide.Len(); i++ {
			single := vSlide.Index(i).Interface()

			switch v := single.(type) {
			case []T:
				result = append(result, Flat[T](v)...)
			case T:
				result = append(result, v)
			}
		}
	}
	return result
}

func Map[T any, R any](slide []T, transformer func(x T) (R, error)) ([]R, error) {
	var newSlide []R

	for _, v := range slide {
		newValue, err := transformer(v)
		if err != nil {
			return nil, err
		}

		newSlide = append(
			newSlide,
			newValue,
		)
	}
	return newSlide, nil
}

func Index[T any](slide []T, cond func(x T) bool) int {
	for i, v := range slide {
		if cond(v) {
			return i
		}
	}
	return -1
}

func Some[T any](slide []T, cond func(x T) bool) bool {
	for _, v := range slide {
		if cond(v) {
			return true
		}
	}
	return false
}
