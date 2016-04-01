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

func unmarshalValue(data []byte, val reflect.Value) int {
   if !val.IsValid() || !val.CanSet() {
        return 0
    }

    v, dt, of, err := Get(data)

    switch val.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        if dt == Number && err == nil {
            val.SetInt(ParseInt(v))
        }
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        if dt == Number && err == nil {
            val.SetUint(uint64(ParseInt(v)))
        }
    case reflect.String:
        if dt == String && err == nil {
            val.SetString(unsafeBytesToString(v))
        }
    case reflect.Bool:
        if dt == Boolean && err == nil {
            val.SetBool(ParseBoolean(v))
        }
    case reflect.Ptr:
        obj := reflect.New(val.Type().Elem())
        unmarshalValue(v, obj.Elem())
        val.Set(obj)
    case reflect.Struct:
        obj := reflect.New(val.Type())
        unmarshalStruct(v, obj.Elem())
        val.Set(obj.Elem())
    case reflect.Slice:
        sT := val.Type().Elem()
        s := reflect.MakeSlice(val.Type(), 0, 0)
        ArrayEach(v, func(value []byte, dataType int, offset int, err error){
            el := reflect.New(sT)

            switch sT.Kind() {
            case reflect.Struct:
                unmarshalStruct(value, el.Elem())
            case reflect.Ptr:
                unmarshalValue(value, el.Elem())
            default:
                unmarshalValue(value, el.Elem())
            }

            s = reflect.Append(s, el.Elem())
        })
        val.Set(s)
    }

    return of
}

func unmarshalStruct(data []byte, val reflect.Value) int {
    sName := val.Type().Name()
    var sCache *structCache
    var ok bool

    // Cache struct info
    if sCache, ok = cache[sName]; !ok {
        count := val.NumField()
        fields := make([][]string, count)
        // fieldTypes := make([]reflect.Kind, count)

        for i := 0; i < val.NumField(); i++ {
            // valueField := val.Field(i)
            typeField := val.Type().Field(i)
            tag := typeField.Tag
            jsonKey := tag.Get("json")

            if jsonKey != "" {
                fields[i] = []string{jsonKey}
            } else {
                fields[i] = []string{strings.ToLower(string(typeField.Name[:1])) + string(typeField.Name[1:])}
            }
            // fieldTypes[i] = valueField.Kind()
        }

        sCache = &structCache{fields: fields}
        cache[sName] = sCache
    }

    fields := sCache.fields
    // fieldTypes := sCache.fieldTypes

    offset := KeyEach(data, func(i int, d []byte) int {
        f := val.Field(i)

        return unmarshalValue(d, f)
    }, fields...)
    // panic(string(data))

    return offset
}

func Unmarshal(data []byte, v interface{}) error {
    val := reflect.ValueOf(v).Elem()

    unmarshalStruct(data, val)

    return nil
}