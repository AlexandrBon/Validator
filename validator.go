package Validator

import (
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

var ErrWrongValue = errors.New("Variable value not validated")
var ErrWrongIn = errors.New("'in' is empty")
var ErrWrongMin = errors.New("min tag is incorrect")
var ErrWrongMax = errors.New("max tag is incorrect")
var ErrWrongMinMax = errors.New("minmax tag is incorrect")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var errMessages []string
	for _, m := range v {
		errMessages = append(errMessages, m.Err.Error())
	}
	return strings.Join(errMessages, ", ")
}

const (
	ValidateTag = "validate"
	Sep         = ":"
)

func getInt(str string) (int, error) {
	if val, err := strconv.Atoi(str); err != nil {
		return 0, err
	} else {
		return val, nil
	}
}

func getField(v any, fName string) reflect.Value {
	return reflect.Indirect(reflect.ValueOf(v)).FieldByName(fName)
}

func Contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func validateInts(strs []string) []int {
	var ints []int
	for _, s := range strs {
		if val, err := getInt(s); err != nil {
			return nil
		} else {
			ints = append(ints, val)
		}
	}
	return ints
}

type Predicate func(value reflect.Value) bool

func validateTagWithPredicate(v any, structFieldName string, pred Predicate) bool {
	f := getField(v, structFieldName)
	return pred(f)
}

func Validate(v any) error {
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return ErrNotStruct
	}

	errs := ValidationErrors{}

	for _, structField := range reflect.VisibleFields(reflect.TypeOf(v)) {
		if !structField.IsExported() && structField.Tag != "" {
			errs = append(errs, ValidationError{Err: ErrValidateForUnexportedFields})
		} else if structField.IsExported() && structField.Tag != "" {
			tagValue := strings.Split(structField.Tag.Get(ValidateTag), Sep)
			switch tagValue[0] {
			case "len":
				requiredLen, err := getInt(tagValue[1])
				if err != nil {
					errs = append(errs, ValidationError{Err: ErrInvalidValidatorSyntax})
					continue
				}
				if !validateTagWithPredicate(v, structField.Name, func(f reflect.Value) bool {
					return len(f.String()) == requiredLen
				}) {
					errs = append(errs, ValidationError{Err: ErrWrongValue})
				}
			case "in":
				valList := strings.Split(tagValue[1], ",")
				if valList[0] == "" {
					errs = append(errs, ValidationError{Err: ErrWrongIn})
					continue
				}

				if !validateTagWithPredicate(v, structField.Name, func(f reflect.Value) bool {
					flag := false
					if f.Kind() == reflect.Int {
						if ints := validateInts(valList); ints != nil && Contains(ints, int(f.Int())) {
							flag = true
						}
					}
					if f.Kind() == reflect.String && Contains(valList, f.String()) {
						flag = true
					}
					return flag
				}) {
					errs = append(errs, ValidationError{Err: ErrWrongValue})
				}
			case "min":
				minVal, err := getInt(tagValue[1])
				if err != nil {
					errs = append(errs, ValidationError{Err: ErrWrongMin})
					continue
				}

				if !validateTagWithPredicate(v, structField.Name, func(f reflect.Value) bool {
					return (f.Kind() == reflect.Int && int(f.Int()) >= minVal) ||
						(f.Kind() == reflect.String && len(f.String()) >= minVal)
				}) {
					errs = append(errs, ValidationError{Err: ErrWrongValue})
				}
			case "max":
				maxVal, err := getInt(tagValue[1])
				if err != nil {
					errs = append(errs, ValidationError{Err: ErrWrongMax})
					continue
				}

				if !validateTagWithPredicate(v, structField.Name, func(f reflect.Value) bool {
					return (f.Kind() == reflect.Int && int(f.Int()) <= maxVal) ||
						(f.Kind() == reflect.String && len(f.String()) <= maxVal)
				}) {
					errs = append(errs, ValidationError{Err: ErrWrongValue})
				}
			case "minmax":
				minAndMaxValues := strings.Split(tagValue[1], ",")
				if len(minAndMaxValues) != 2 {
					errs = append(errs, ValidationError{Err: ErrWrongMinMax})
					return errs
				}
				minVal, err := getInt(minAndMaxValues[0])
				maxVal, err := getInt(minAndMaxValues[1])
				if err != nil {
					errs = append(errs, ValidationError{Err: ErrWrongMinMax})
					continue
				}

				if !validateTagWithPredicate(v, structField.Name, func(f reflect.Value) bool {
					return (f.Kind() == reflect.Int && int(f.Int()) >= minVal && int(f.Int()) <= maxVal) ||
						(f.Kind() == reflect.String && len(f.String()) >= minVal && len(f.String()) <= maxVal)
				}) {
					errs = append(errs, ValidationError{Err: ErrWrongValue})
				}
			}

		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
