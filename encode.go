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

    offsets := KeyOffsets(data, fields...)

    for i, of := range offsets {
        f := val.Field(i)

        if !f.IsValid() || !f.CanSet() {
            continue
        }

        switch fieldTypes[i] {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            v, err := GetInt(data[of:])

            if err == nil {
                f.SetInt(v)
            }
        case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
            v, err := GetInt(data[of:])

            if err == nil {
                f.SetUint(uint64(v))
            }
        case reflect.String:
            v, err := GetString(data[of:])

            if err == nil {
                f.SetString(v)
            }
        case reflect.Bool:
            v, err := GetBoolean(data[of:])

            if err == nil {
                f.SetBool(v)
            }
        }
    }

    return nil
}