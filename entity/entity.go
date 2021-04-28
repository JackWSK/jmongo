package entity

import (
    "fmt"
    "github.com/pkg/errors"
    "jmongo/errortype"
    "jmongo/utils"
    "reflect"
    "sync"
)

var cacheStore = &sync.Map{}

type Entity struct {
    Name                string
    ModelType           reflect.Type
    Collection          string
    PrimaryField        *EntityField
    DBNames             []string
    PrimaryFields       []*EntityField
    PrimaryFieldDBNames []string
    Fields              []*EntityField
    AllFields           []*EntityField
    FieldsByName        map[string]*EntityField
    FieldsByDBName      map[string]*EntityField
}

// get data type from dialector
func newEntity(dest interface{}) (*Entity, error) {

    if dest == nil {
        return nil, errors.WithStack(fmt.Errorf("%w: %s", errortype.ErrUnsupportedDataType, "dest is nil"))
    }

    modelType := reflect.ValueOf(dest).Type()

    return newEntityByModelType(modelType, nil)
}

func newEntityByModelType(modelType reflect.Type, index []int) (*Entity, error) {

    for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
        modelType = modelType.Elem()
    }

    if modelType.Kind() != reflect.Struct {
        if modelType.PkgPath() == "" {
            return nil, errors.WithStack(fmt.Errorf("%w: %+v", errortype.ErrUnsupportedDataType, modelType.Name()))
        }
        return nil, errors.WithStack(fmt.Errorf("%w: %v.%v", errortype.ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name()))
    }

    if v, ok := cacheStore.Load(modelType); ok {
        s := v.(*Entity)
        return s, nil
    }

    // get collection name for model
    modelValue := reflect.New(modelType)
    var collectionName string
    if tabler, ok := modelValue.Interface().(CollectionNameSupplier); ok {
        collectionName = tabler.CollectionName()
    } else {

        collectionName = utils.LowerFirst(modelType.Name())
    }

    entity := &Entity{}

    // extract fields from model type
    fields, allFields, err := extractFields(modelType, index)
    if err != nil {
        return nil, err
    }

    // extract id field from fields
    idField := extractIdField(allFields)
    if idField == nil {
        return nil, errortype.ErrIdFieldDoesNotExists
    }

    // create map for fields by name and by db name
    fieldsByName, fieldsByDBName := makeFieldsByNameAndByDBName(fields)

    // entity
    entity.Name = modelType.Name()
    entity.ModelType = modelType
    entity.Fields = fields
    entity.AllFields = fields
    entity.Collection = collectionName
    entity.FieldsByName = fieldsByName
    entity.FieldsByDBName = fieldsByDBName

    return entity, nil
}

func extractFields(modelType reflect.Type, index []int) (fields []*EntityField, allFields []*EntityField, err error) {

    // get field
    for i := 0; i < modelType.NumField(); i++ {
        // clone index
        cloneIndex := make([]int, len(index), len(index)+1)
        copy(cloneIndex, index)
        cloneIndex = append(cloneIndex, i)

        structField := modelType.Field(i)
        tag := structField.Tag.Get("bson")

        // parse to get bson info
        structTags, err := parseTags(utils.LowerFirst(structField.Name), tag)
        if err != nil {
            return nil, nil, err
        }

        // filter skip field
        if structTags.Skip {
            continue
        }

        field, err := newField(structField, structTags, cloneIndex)
        if err != nil {
            return nil, nil, err
        }
        fields = append(fields, field)
        if field.Entity != nil {
            allFields = append(allFields, field.Entity.Fields...)
        } else {
            allFields = append(allFields, field)
        }
    }

    return fields, allFields, nil
}

func extractIdField(fields []*EntityField) *EntityField {

    var idField *EntityField
    for _, field := range fields {
        if field.Entity != nil {
            idField = extractIdField(field.Entity.Fields)
            if idField != nil {
                break
            }
        } else {
            if field.Id {
                idField = field
                break
            }
        }
    }

    return idField
}

func makeFieldsByNameAndByDBName(fields []*EntityField) (fieldsByName, fieldsByDBName map[string]*EntityField) {
    fieldsByName = map[string]*EntityField{}
    fieldsByDBName = map[string]*EntityField{}

    for _, field := range fields {

        if v, ok := fieldsByDBName[field.DBName]; !ok {
            fieldsByDBName[field.DBName] = v
        }

        if v, ok := fieldsByName[field.Name]; !ok {
            fieldsByName[field.Name] = v
        }
    }

    return fieldsByName, fieldsByDBName
}

func (th *Entity) MakeSlice() reflect.Value {
    slice := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(th.ModelType)), 0, 20)
    results := reflect.New(slice.Type())
    results.Elem().Set(slice)
    return results
}

func (th *Entity) LookUpField(name string) *EntityField {
    if field, ok := th.FieldsByDBName[name]; ok {
        return field
    }
    if field, ok := th.FieldsByName[name]; ok {
        return field
    }
    return nil
}

func (th *Entity) PrimaryKeyDBName() string {
    if th.PrimaryField != nil {
        return th.PrimaryField.DBName
    }
    return "_id"
}

var mutex sync.Mutex

func GetModelType(dest interface{}) reflect.Type {
    modelType := reflect.ValueOf(dest).Type()
    for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
        modelType = modelType.Elem()
    }
    return modelType
}


func GetOrParse(dest interface{}) (entity *Entity, err error) {

    modelType := GetModelType(dest)

    if modelType.Kind() != reflect.Struct {
        if modelType.PkgPath() == "" {
            return nil, errors.WithStack(fmt.Errorf("%w: %+v", errortype.ErrUnsupportedDataType, dest))
        }
        return nil, errors.WithStack(fmt.Errorf("%w: %v.%v", errortype.ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name()))
    }

    if v, ok := cacheStore.Load(modelType); ok {
        return v.(*Entity), nil
    }

    mutex.Lock()
    defer func() {
        mutex.Unlock()
    }()
    if _, ok := cacheStore.Load(modelType); !ok {
        entity, err = newEntity(dest)
        if err != nil {
            return nil, err
        }
        cacheStore.Store(modelType, entity)
    }

    return entity, nil
}
