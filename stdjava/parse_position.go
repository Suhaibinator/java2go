package stdjava

type ParsePosition struct{ index, errorIndex int32 }

func NewParsePosition(index int32) *ParsePosition {
	return &ParsePosition{index: index, errorIndex: -1}
}
func (position *ParsePosition) GetIndex() int32 {
	ReferenceRequireNonNull(position)
	return position.index
}
func (position *ParsePosition) SetIndex(index int32) {
	ReferenceRequireNonNull(position)
	position.index = index
}
func (position *ParsePosition) GetErrorIndex() int32 {
	ReferenceRequireNonNull(position)
	return position.errorIndex
}
func (position *ParsePosition) SetErrorIndex(index int32) {
	ReferenceRequireNonNull(position)
	position.errorIndex = index
}
func (*ParsePosition) JavaDynamicTypeID() TypeID { return "java.text.ParsePosition" }
func init()                                      { RegisterJavaType("java.text.ParsePosition", ObjectTypeID) }
