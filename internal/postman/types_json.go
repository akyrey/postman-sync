package postman

// JSON (un)marshalling that keeps unmodelled fields in each type's Extra map.
// Each alias type has the same fields but none of the methods, which avoids
// infinite recursion when delegating to encoding/json.

type (
	collectionFields       Collection
	infoFields             Info
	collectionItemFields   CollectionItem
	requestFields          Request
	urlFields              URL
	urlParamFields         URLParam
	headerFields           Header
	bodyFields             Body
	formParamFields        FormParam
	authFields             Auth
	authAttributeFields    AuthAttribute
	eventFields            Event
	scriptFields           Script
	variableFields         Variable
	environmentFields      Environment
	environmentValueFields EnvironmentValue
)

func (v *Collection) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*collectionFields)(v), &v.Extra)
}

func (v Collection) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(collectionFields(v), v.Extra)
}

func (v *Info) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*infoFields)(v), &v.Extra)
}

func (v Info) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(infoFields(v), v.Extra)
}

func (v *CollectionItem) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*collectionItemFields)(v), &v.Extra)
}

func (v CollectionItem) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(collectionItemFields(v), v.Extra)
}

func (v *Request) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*requestFields)(v), &v.Extra)
}

func (v Request) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(requestFields(v), v.Extra)
}

func (v *URL) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*urlFields)(v), &v.Extra)
}

func (v URL) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(urlFields(v), v.Extra)
}

func (v *URLParam) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*urlParamFields)(v), &v.Extra)
}

func (v URLParam) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(urlParamFields(v), v.Extra)
}

func (v *Header) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*headerFields)(v), &v.Extra)
}

func (v Header) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(headerFields(v), v.Extra)
}

func (v *Body) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*bodyFields)(v), &v.Extra)
}

func (v Body) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(bodyFields(v), v.Extra)
}

func (v *FormParam) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*formParamFields)(v), &v.Extra)
}

func (v FormParam) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(formParamFields(v), v.Extra)
}

func (v *Auth) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*authFields)(v), &v.Extra)
}

func (v Auth) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(authFields(v), v.Extra)
}

func (v *AuthAttribute) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*authAttributeFields)(v), &v.Extra)
}

func (v AuthAttribute) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(authAttributeFields(v), v.Extra)
}

func (v *Event) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*eventFields)(v), &v.Extra)
}

func (v Event) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(eventFields(v), v.Extra)
}

func (v *Script) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*scriptFields)(v), &v.Extra)
}

func (v Script) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(scriptFields(v), v.Extra)
}

func (v *Variable) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*variableFields)(v), &v.Extra)
}

func (v Variable) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(variableFields(v), v.Extra)
}

func (v *Environment) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*environmentFields)(v), &v.Extra)
}

func (v Environment) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(environmentFields(v), v.Extra)
}

func (v *EnvironmentValue) UnmarshalJSON(b []byte) error {
	return unmarshalWithExtra(b, (*environmentValueFields)(v), &v.Extra)
}

func (v EnvironmentValue) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(environmentValueFields(v), v.Extra)
}
