package stamp

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

var (
	scopeTypePattern   = regexp.MustCompile(`Scope\[(.*)]`)
	embeddableMdlTypes = []reflect.Type{
		reflect.TypeOf(&Model[Root]{}),
		reflect.TypeOf(&Scope[Root]{}),
		reflect.TypeFor[modelWrapper](),
		reflect.TypeFor[Generator[any]](),
	}
)

type (
	ModelSet struct {
		typeIdx             map[modelType]struct{}
		typeByQualifiedName map[string]modelType
		childTypes          map[modelType][]modelType
		parentType          map[modelType]modelType
		markerIdx           map[modelType][]Marker
		callbackFuncIdx     map[modelType]map[string]func(modelWrapper, routableAction) func(reflect.Value)
	}
	registerModel interface {
		modelWrapper
	}
)

func NewModelSet() *ModelSet {
	return &ModelSet{
		typeIdx:             make(map[modelType]struct{}),
		typeByQualifiedName: map[string]modelType{rootTypeName: rootType},
		childTypes:          make(map[modelType][]modelType),
		parentType:          make(map[modelType]modelType),
		markerIdx:           make(map[modelType][]Marker),
		callbackFuncIdx:     make(map[modelType]map[string]func(modelWrapper, routableAction) func(reflect.Value)),
	}
}

// TODO: check that struct has no fields at all (we don't want any state there)
// TODO: type check parent matches model somehow?
// TODO: fail when 2 params of a handler have the same type
// TODO: check if handler contains unexpected parameter types (ie from outside of this context)
func RegisterModel[M registerModel](set *ModelSet) {
	ptrType := reflect.TypeFor[M]()
	elemType := ptrType.Elem()
	mdlType := modelType{ptrType: ptrType, structType: elemType, name: elemType.Name()}

	if _, ok := set.typeIdx[mdlType]; ok {
		panic(fmt.Sprintf("%q already registered", mdlType.name))
	}
	set.typeIdx[mdlType] = struct{}{}
	set.typeByQualifiedName[qualifiedTypeName(mdlType.structType)] = mdlType

	for i := 0; i < mdlType.structType.NumField(); i++ {
		field := mdlType.structType.Field(i)
		fieldTypeName := field.Type.String()
		parentMatch := scopeTypePattern.FindStringSubmatch(fieldTypeName)
		if len(parentMatch) == 2 {
			scopeFieldMdlTypeName := strings.TrimPrefix(parentMatch[1], "*")
			scopeFieldMdlType, ok := set.typeByQualifiedName[scopeFieldMdlTypeName]
			if !ok {
				panic(fmt.Sprintf("scope %q from model %q must be a registered model", scopeFieldMdlTypeName, mdlType.name))
			}
			set.childTypes[scopeFieldMdlType] = append(set.childTypes[scopeFieldMdlType], mdlType)
			set.parentType[mdlType] = scopeFieldMdlType
		}
	}

	// register model handlers
	// TODO: scan unexported methods, too, in case user accidentally made one private
	//mdlInst := reflect.New(mdlType.structType).Interface().(modelWrapper)
	//mdlInstVal := reflect.ValueOf(mdlInst)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		methodType := method.Type
		if isFromEmbedded(method) {
			continue
		}

		switch {
		// ignore model getters
		case methodType.NumIn() == 1 && methodType.NumOut() == 1 && strings.HasPrefix(method.Name, "Get"):
			// TODO: check that return type is a model
			continue

		// index action handlers
		case methodType.NumIn() == 2 && methodType.NumOut() == 1 && methodType.Out(0).Kind() == reflect.Func:
			if _, ok := set.callbackFuncIdx[mdlType]; !ok {
				set.callbackFuncIdx[mdlType] = make(map[string]func(modelWrapper, routableAction) func(reflect.Value))
			}
			inputType := methodType.In(1)
			set.callbackFuncIdx[mdlType][inputType.String()] = func(mw modelWrapper, action routableAction) func(reflect.Value) {
				ret := method.Func.Call([]reflect.Value{reflect.ValueOf(mw), reflect.ValueOf(action)})
				return func(v reflect.Value) { ret[0].Call([]reflect.Value{v}) }
			}

		// index rules
		//case method.Type.NumOut() == 1 && method.Type.Out(0).Implements(propType):
		//	if method.Type.NumIn() != 1 {
		//		panic(fmt.Sprintf("property %q on %q must not have any parameters", method.Name, mdlType.name))
		//	}
		//	if method.Type.NumOut() != 1 {
		//		panic(fmt.Sprintf("property %q on %q must return `Prop` or `Rule`", method.Name, mdlType.name))
		//	}
		//	newProp := method.Func.Call([]reflect.Value{mdlInstVal})[0].Interface().(prop)
		//	newProp.setName(fmt.Sprintf("%s.%s", mdlType.name, method.Name))
		//	if err := newProp.Validate(); err != nil {
		//		panic(fmt.Sprintf("property %q on %q failed validation: %v", method.Name, mdlType.name, err))
		//	}
		//	if method.Type.Out(0) == ruleType {
		//		set.ruleIdx[mdlType] = append(set.ruleIdx[mdlType], func(mw modelWrapper) (prop, any, error) {
		//			res, err := newProp.eval(mw.getPropCtx())
		//			return newProp, res, err
		//		})
		//	}

		default:
			panic(fmt.Sprintf("method %q on %q is not a getter or an action handler", method.Name, mdlType.name))
		}
	}

	// register model markers
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		switch {
		case field.Type.AssignableTo(markerType):
			set.markerIdx[mdlType] = append(set.markerIdx[mdlType], Marker{
				name: field.Name,
				// TODO: parse doc tag
			})
		}
	}
}

func (s ModelSet) newModel(
	env modelEnv,
	id ID,
	mdlType modelType,
	scope modelWrapper,
) modelWrapper {
	mdl := &internalModel{
		mdlEnv:  env,
		typeOf:  mdlType,
		propCtx: newPropContext(),
	}
	mdl.updateIdentity(scope.getKey(), id)

	v := reflect.New(mdlType.structType)
	mw := v.Interface().(modelWrapper)
	mw.setModel(mdl)
	mw.setScope(scope)
	for _, marker := range s.markerIdx[mdlType] {
		markerCopy := marker.copy()
		markerCopy.owner = mw
		v.Elem().FieldByName(marker.name).Set(reflect.ValueOf(markerCopy))
	}
	return mw
}

func (s ModelSet) consume(mw modelWrapper, action routableAction) func(reflect.Value) {
	mdlCallbacks := s.callbackFuncIdx[mw.getType()]
	if mdlCallbacks == nil {
		return nil
	}
	actionTypeStr := fmt.Sprintf("%T", action)
	actionCallback := mdlCallbacks[actionTypeStr]
	if actionCallback == nil {
		return nil
	}
	return actionCallback(mw, action)
}

func (s ModelSet) validate(a any) {
	if reflect.TypeOf(a).Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected struct, got %T", a))
	}
	err := validator.Struct(a)
	if err != nil {
		// TODO: use logger
		panic(fmt.Sprintf("%T failed validation: %v", a, err))
	}
}

func (s ModelSet) childTypesOf(m modelWrapper) []modelType {
	return s.childTypes[m.getType()]
}

func (s ModelSet) pathTo(ty reflect.Type) []modelType {
	typeName := qualifiedTypeName(ty)
	curType, ok := s.typeByQualifiedName[typeName]
	if !ok {
		panic(fmt.Sprintf("type %q is not a registered model", ty))
	}

	var res []modelType
	for curType != rootType {
		res = append(res, curType)
		curType = s.parentType[curType]
	}
	slices.Reverse(res)
	return res
}

func isFromEmbedded(method reflect.Method) bool {
	for _, typ := range embeddableMdlTypes {
		if _, ok := typ.MethodByName(method.Name); ok {
			return true
		}
		if typ.Kind() == reflect.Ptr {
			if _, ok := typ.Elem().MethodByName(method.Name); ok {
				return true
			}
		}
	}
	return false
}
