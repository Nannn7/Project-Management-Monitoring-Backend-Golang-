package validation

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func IsUUID(s string) bool { return uuidRe.MatchString(s) }

var validate = func() *validator.Validate {
	v := validator.New()
	// pakai nama field JSON ("directorateId"), bukan nama struct Go ("DirectorateID")
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	return v
}()

// Fields mengembalikan map field->pesan, atau nil kalau valid.
func Fields(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return map[string]string{"_": "Data tidak valid"}
	}
	out := make(map[string]string, len(ve))
	for _, fe := range ve {
		out[fe.Field()] = message(fe)
	}
	return out
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "Wajib diisi"
	case "email":
		return "Format email tidak valid"
	case "uuid":
		return "Format UUID tidak valid"
	case "min":
		unit := "karakter"
		if fe.Kind() == reflect.Slice {
			unit = "item"
		}
		return fmt.Sprintf("Minimal %s %s", fe.Param(), unit)
	default:
		return "Data tidak valid"
	}
}
