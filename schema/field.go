package schema

import (
    "reflect"
)

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {

    dbName := schema.fieldDBName(fieldStruct)

    field := &Field{
        Name:        fieldStruct.Name,
        DBName:      dbName,
        FieldType:   fieldStruct.Type,
        StructField: fieldStruct,
        Tag:         fieldStruct.Tag,
        Schema:      schema,
    }

    field.setupValuerAndSetter()
    return field
}

func (schema *Schema) fieldTagSetting(fieldStruct reflect.StructField) map[string]string {
    if name, ok := fieldStruct.Tag.Lookup("jmongo"); ok && name != "" {
        return ParseTagSetting("jmongo", ";")
    }
    return map[string]string{}
}

func (schema *Schema) fieldDBName(fieldStruct reflect.StructField) string {
    if name, ok := fieldStruct.Tag.Lookup("bson"); ok && name != "" {
        return name
    } else {
        return fieldStruct.Name
    }
}

func (schema *Schema) fieldIsObjectId(fieldStruct reflect.StructField) string {
    if name, ok := fieldStruct.Tag.Lookup("bson"); ok && name != "" {
        return name
    } else {
        return fieldStruct.Name
    }
}

//func (schema *Schema) isPrimaryKey(fieldStruct reflect.StructField) bool {
//    //if fieldStruct.Tag.Lookup("") {
//    //
//    //}
//}

type Field struct {
    Name           string
    DBName         string
    PrimaryKey     bool
    FieldType      reflect.Type
    StructField    reflect.StructField
    Tag            reflect.StructTag
    TagSettings    map[string]string
    Schema         *Schema
    EmbeddedSchema *Schema
    OwnerSchema    *Schema
    ReflectValueOf func(reflect.Value) reflect.Value
    ValueOf        func(reflect.Value) (value interface{}, zero bool)
}

// create valuer, setter when parse struct
func (field *Field) setupValuerAndSetter() {
    // ValueOf
    switch {
    case len(field.StructField.Index) == 1:
        field.ValueOf = func(value reflect.Value) (interface{}, bool) {
            fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0])
            return fieldValue.Interface(), fieldValue.IsZero()
        }
    case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0:
        field.ValueOf = func(value reflect.Value) (interface{}, bool) {
            fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0]).Field(field.StructField.Index[1])
            return fieldValue.Interface(), fieldValue.IsZero()
        }
    default:
        field.ValueOf = func(value reflect.Value) (interface{}, bool) {
            v := reflect.Indirect(value)

            for _, idx := range field.StructField.Index {
                if idx >= 0 {
                    v = v.Field(idx)
                } else {
                    v = v.Field(-idx - 1)

                    if v.Type().Elem().Kind() == reflect.Struct {
                        if !v.IsNil() {
                            v = v.Elem()
                        } else {
                            return nil, true
                        }
                    } else {
                        return nil, true
                    }
                }
            }
            return v.Interface(), v.IsZero()
        }
    }

    // ReflectValueOf
    switch {
    case len(field.StructField.Index) == 1:
        if field.FieldType.Kind() == reflect.Ptr {
            field.ReflectValueOf = func(value reflect.Value) reflect.Value {
                fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0])
                return fieldValue
            }
        } else {
            field.ReflectValueOf = func(value reflect.Value) reflect.Value {
                return reflect.Indirect(value).Field(field.StructField.Index[0])
            }
        }
    case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0 && field.FieldType.Kind() != reflect.Ptr:
        field.ReflectValueOf = func(value reflect.Value) reflect.Value {
            return reflect.Indirect(value).Field(field.StructField.Index[0]).Field(field.StructField.Index[1])
        }
    default:
        field.ReflectValueOf = func(value reflect.Value) reflect.Value {
            v := reflect.Indirect(value)
            for idx, fieldIdx := range field.StructField.Index {
                if fieldIdx >= 0 {
                    v = v.Field(fieldIdx)
                } else {
                    v = v.Field(-fieldIdx - 1)
                }

                if v.Kind() == reflect.Ptr {
                    if v.Type().Elem().Kind() == reflect.Struct {
                        if v.IsNil() {
                            v.Set(reflect.New(v.Type().Elem()))
                        }
                    }

                    if idx < len(field.StructField.Index)-1 {
                        v = v.Elem()
                    }
                }
            }
            return v
        }
    }

    //fallbackSetter := func(value reflect.Value, v interface{}, setter func(reflect.Value, interface{}) error) (err error) {
    //    if v == nil {
    //        field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //    } else {
    //        reflectV := reflect.ValueOf(v)
    //
    //        if reflectV.Type().AssignableTo(field.FieldType) {
    //            field.ReflectValueOf(value).Set(reflectV)
    //            return
    //        } else if reflectV.Type().ConvertibleTo(field.FieldType) {
    //            field.ReflectValueOf(value).Set(reflectV.Convert(field.FieldType))
    //            return
    //        } else if field.FieldType.Kind() == reflect.Ptr {
    //            fieldValue := field.ReflectValueOf(value)
    //
    //            if reflectV.Type().AssignableTo(field.FieldType.Elem()) {
    //                if !fieldValue.IsValid() {
    //                    fieldValue = reflect.New(field.FieldType.Elem())
    //                } else if fieldValue.IsNil() {
    //                    fieldValue.Set(reflect.New(field.FieldType.Elem()))
    //                }
    //                fieldValue.Elem().Set(reflectV)
    //                return
    //            } else if reflectV.Type().ConvertibleTo(field.FieldType.Elem()) {
    //                if fieldValue.IsNil() {
    //                    fieldValue.Set(reflect.New(field.FieldType.Elem()))
    //                }
    //
    //                fieldValue.Elem().Set(reflectV.Convert(field.FieldType.Elem()))
    //                return
    //            }
    //        }
    //
    //        if reflectV.Kind() == reflect.Ptr {
    //            if reflectV.IsNil() {
    //                field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //            } else {
    //                err = setter(value, reflectV.Elem().Interface())
    //            }
    //        } else if valuer, ok := v.(driver.Valuer); ok {
    //            if v, err = valuer.Value(); err == nil {
    //                err = setter(value, v)
    //            }
    //        } else {
    //            return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
    //        }
    //    }
    //
    //    return
    //}
    //
    //// Set
    //switch field.FieldType.Kind() {
    //case reflect.Bool:
    //    field.Set = func(value reflect.Value, v interface{}) error {
    //        switch data := v.(type) {
    //        case bool:
    //            field.ReflectValueOf(value).SetBool(data)
    //        case *bool:
    //            if data != nil {
    //                field.ReflectValueOf(value).SetBool(*data)
    //            } else {
    //                field.ReflectValueOf(value).SetBool(false)
    //            }
    //        case int64:
    //            if data > 0 {
    //                field.ReflectValueOf(value).SetBool(true)
    //            } else {
    //                field.ReflectValueOf(value).SetBool(false)
    //            }
    //        case string:
    //            b, _ := strconv.ParseBool(data)
    //            field.ReflectValueOf(value).SetBool(b)
    //        default:
    //            return fallbackSetter(value, v, field.Set)
    //        }
    //        return nil
    //    }
    //case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
    //    field.Set = func(value reflect.Value, v interface{}) (err error) {
    //        switch data := v.(type) {
    //        case int64:
    //            field.ReflectValueOf(value).SetInt(data)
    //        case int:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case int8:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case int16:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case int32:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case uint:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case uint8:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case uint16:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case uint32:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case uint64:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case float32:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case float64:
    //            field.ReflectValueOf(value).SetInt(int64(data))
    //        case []byte:
    //            return field.Set(value, string(data))
    //        case string:
    //            if i, err := strconv.ParseInt(data, 0, 64); err == nil {
    //                field.ReflectValueOf(value).SetInt(i)
    //            } else {
    //                return err
    //            }
    //        case time.Time:
    //            field.ReflectValueOf(value).SetInt(data.Unix())
    //        case *time.Time:
    //            if data != nil {
    //                field.ReflectValueOf(value).SetInt(data.Unix())
    //            } else {
    //                field.ReflectValueOf(value).SetInt(0)
    //            }
    //        default:
    //            return fallbackSetter(value, v, field.Set)
    //        }
    //        return err
    //    }
    //case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
    //    field.Set = func(value reflect.Value, v interface{}) (err error) {
    //        switch data := v.(type) {
    //        case uint64:
    //            field.ReflectValueOf(value).SetUint(data)
    //        case uint:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case uint8:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case uint16:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case uint32:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case int64:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case int:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case int8:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case int16:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case int32:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case float32:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case float64:
    //            field.ReflectValueOf(value).SetUint(uint64(data))
    //        case []byte:
    //            return field.Set(value, string(data))
    //        case time.Time:
    //            field.ReflectValueOf(value).SetUint(uint64(data.Unix()))
    //        case string:
    //            if i, err := strconv.ParseUint(data, 0, 64); err == nil {
    //                field.ReflectValueOf(value).SetUint(i)
    //            } else {
    //                return err
    //            }
    //        default:
    //            return fallbackSetter(value, v, field.Set)
    //        }
    //        return err
    //    }
    //case reflect.Float32, reflect.Float64:
    //    field.Set = func(value reflect.Value, v interface{}) (err error) {
    //        switch data := v.(type) {
    //        case float64:
    //            field.ReflectValueOf(value).SetFloat(data)
    //        case float32:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case int64:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case int:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case int8:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case int16:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case int32:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case uint:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case uint8:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case uint16:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case uint32:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case uint64:
    //            field.ReflectValueOf(value).SetFloat(float64(data))
    //        case []byte:
    //            return field.Set(value, string(data))
    //        case string:
    //            if i, err := strconv.ParseFloat(data, 64); err == nil {
    //                field.ReflectValueOf(value).SetFloat(i)
    //            } else {
    //                return err
    //            }
    //        default:
    //            return fallbackSetter(value, v, field.Set)
    //        }
    //        return err
    //    }
    //case reflect.String:
    //    field.Set = func(value reflect.Value, v interface{}) (err error) {
    //        switch data := v.(type) {
    //        case string:
    //            field.ReflectValueOf(value).SetString(data)
    //        case []byte:
    //            field.ReflectValueOf(value).SetString(string(data))
    //        case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
    //            field.ReflectValueOf(value).SetString(utils.ToString(data))
    //        case float64, float32:
    //            field.ReflectValueOf(value).SetString(fmt.Sprintf("%."+strconv.Itoa(field.Precision)+"f", data))
    //        default:
    //            return fallbackSetter(value, v, field.Set)
    //        }
    //        return err
    //    }
    //default:
    //    fieldValue := reflect.New(field.FieldType)
    //    switch fieldValue.Elem().Interface().(type) {
    //    case time.Time:
    //        field.Set = func(value reflect.Value, v interface{}) error {
    //            switch data := v.(type) {
    //            case time.Time:
    //                field.ReflectValueOf(value).Set(reflect.ValueOf(v))
    //            case *time.Time:
    //                if data != nil {
    //                    field.ReflectValueOf(value).Set(reflect.ValueOf(data).Elem())
    //                } else {
    //                    field.ReflectValueOf(value).Set(reflect.ValueOf(time.Time{}))
    //                }
    //            case string:
    //                if t, err := now.Parse(data); err == nil {
    //                    field.ReflectValueOf(value).Set(reflect.ValueOf(t))
    //                } else {
    //                    return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
    //                }
    //            default:
    //                return fallbackSetter(value, v, field.Set)
    //            }
    //            return nil
    //        }
    //    case *time.Time:
    //        field.Set = func(value reflect.Value, v interface{}) error {
    //            switch data := v.(type) {
    //            case time.Time:
    //                fieldValue := field.ReflectValueOf(value)
    //                if fieldValue.IsNil() {
    //                    fieldValue.Set(reflect.New(field.FieldType.Elem()))
    //                }
    //                fieldValue.Elem().Set(reflect.ValueOf(v))
    //            case *time.Time:
    //                field.ReflectValueOf(value).Set(reflect.ValueOf(v))
    //            case string:
    //                if t, err := now.Parse(data); err == nil {
    //                    fieldValue := field.ReflectValueOf(value)
    //                    if fieldValue.IsNil() {
    //                        if v == "" {
    //                            return nil
    //                        }
    //                        fieldValue.Set(reflect.New(field.FieldType.Elem()))
    //                    }
    //                    fieldValue.Elem().Set(reflect.ValueOf(t))
    //                } else {
    //                    return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
    //                }
    //            default:
    //                return fallbackSetter(value, v, field.Set)
    //            }
    //            return nil
    //        }
    //    default:
    //        if _, ok := fieldValue.Elem().Interface().(sql.Scanner); ok {
    //            // pointer scanner
    //            field.Set = func(value reflect.Value, v interface{}) (err error) {
    //                reflectV := reflect.ValueOf(v)
    //                if !reflectV.IsValid() {
    //                    field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //                } else if reflectV.Type().AssignableTo(field.FieldType) {
    //                    field.ReflectValueOf(value).Set(reflectV)
    //                } else if reflectV.Kind() == reflect.Ptr {
    //                    if reflectV.IsNil() || !reflectV.IsValid() {
    //                        field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //                    } else {
    //                        return field.Set(value, reflectV.Elem().Interface())
    //                    }
    //                } else {
    //                    fieldValue := field.ReflectValueOf(value)
    //                    if fieldValue.IsNil() {
    //                        fieldValue.Set(reflect.New(field.FieldType.Elem()))
    //                    }
    //
    //                    if valuer, ok := v.(driver.Valuer); ok {
    //                        v, _ = valuer.Value()
    //                    }
    //
    //                    err = fieldValue.Interface().(sql.Scanner).Scan(v)
    //                }
    //                return
    //            }
    //        } else if _, ok := fieldValue.Interface().(sql.Scanner); ok {
    //            // struct scanner
    //            field.Set = func(value reflect.Value, v interface{}) (err error) {
    //                reflectV := reflect.ValueOf(v)
    //                if !reflectV.IsValid() {
    //                    field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //                } else if reflectV.Type().AssignableTo(field.FieldType) {
    //                    field.ReflectValueOf(value).Set(reflectV)
    //                } else if reflectV.Kind() == reflect.Ptr {
    //                    if reflectV.IsNil() || !reflectV.IsValid() {
    //                        field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
    //                    } else {
    //                        return field.Set(value, reflectV.Elem().Interface())
    //                    }
    //                } else {
    //                    if valuer, ok := v.(driver.Valuer); ok {
    //                        v, _ = valuer.Value()
    //                    }
    //
    //                    err = field.ReflectValueOf(value).Addr().Interface().(sql.Scanner).Scan(v)
    //                }
    //                return
    //            }
    //        } else {
    //            field.Set = func(value reflect.Value, v interface{}) (err error) {
    //                return fallbackSetter(value, v, field.Set)
    //            }
    //        }
    //    }
    //}
}
