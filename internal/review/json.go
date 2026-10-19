package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// MaxJSONBytes bounds untrusted policy and attestation inputs before decoding.
const MaxJSONBytes = 1 << 20

func decodeJSON(data []byte, dst any) error {
	if len(data) == 0 || len(data) > MaxJSONBytes {
		return fmt.Errorf("JSON input must contain 1..%d bytes", MaxJSONBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := checkValue(decoder, reflect.TypeOf(dst).Elem(), 0); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("JSON must contain exactly one value")
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("invalid JSON value: %w", err)
	}
	return nil
}

func checkValue(decoder *json.Decoder, typ reflect.Type, depth int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting exceeds 32")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("null is not permitted")
	}
	switch typ.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return fmt.Errorf("expected object")
		}
		if err := checkObject(decoder, typ, depth); err != nil {
			return err
		}
	case reflect.Slice:
		if token != json.Delim('[') {
			return fmt.Errorf("expected array")
		}
		for decoder.More() {
			if err := checkValue(decoder, typ.Elem(), depth+1); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		if _, compound := token.(json.Delim); compound {
			return fmt.Errorf("expected scalar")
		}
	}
	return nil
}

func checkObject(decoder *json.Decoder, typ reflect.Type, depth int) error {
	fields := map[string]reflect.Type{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fields[strings.Split(field.Tag.Get("json"), ",")[0]] = field.Type
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		field, known := fields[key]
		if !ok || !known || seen[key] {
			return fmt.Errorf("unknown or duplicate field %q", key)
		}
		seen[key] = true
		if err := checkValue(decoder, field, depth+1); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	_, err := decoder.Token()
	return err
}
