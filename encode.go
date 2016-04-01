package jsonparser

import (
    "reflect"
    "strings"
    _ "fmt"
)

type structCache struct {
    fields [][]string
    fieldTypes []reflect.Kind
}

var cache map[string]*structCache

func init() {
    cache = make(map[string]*structCache)
}

func Unmarshal(data []byte, v interface{}) error {
    val := reflect.ValueOf(v).Elem()

    sName := val.Type().Name()
    var sCache *structCache
    var ok bool

    // Cache struct info
    if sCache, ok = cache[sName]; !ok {
        count := val.NumField()
        fields := make([][]string, count)
        fieldTypes := make([]reflect.Kind, count)

        for i := 0; i < val.NumField(); i++ {
            valueField := val.Field(i)
            typeField := val.Type().Field(i)
            tag := typeField.Tag
            jsonKey := tag.Get("json")

            if jsonKey != "" {
                fields[i] = []string{jsonKey}
            } else {
                fields[i] = []string{strings.ToLower(typeField.Name)}
            }
            fieldTypes[i] = valueField.Kind()
        }

        sCache = &structCache{fields, fieldTypes}
        cache[sName] = sCache
    }

    fields := sCache.fields
    fieldTypes := sCache.fieldTypes

    KeyEach(data, func(i int, d []byte) int {
        f := val.Field(i)
        if !f.IsValid() || !f.CanSet() {
            return 0
        }

        v, dt, of, err := Get(d)

        switch fieldTypes[i] {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            if dt == Number && err == nil {
                f.SetInt(ParseInt(v))
            }
        case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
            if dt == Number && err == nil {
                f.SetUint(uint64(ParseInt(v)))
            }
        case reflect.String:
            if dt == String && err == nil {
                f.SetString(unsafeBytesToString(v))
            }
        case reflect.Bool:
            if dt == Boolean && err == nil {
                f.SetBool(ParseBoolean(v))
            }
        }

        return of
    }, fields...)

    return nil
}