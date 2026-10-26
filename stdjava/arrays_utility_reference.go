package stdjava

// JavaArrays is the nominal reference type of java.util.Arrays. Java supplies
// no public constructor. The nonzero carrier retains normal pointer identity
// without adding a factory or instance behavior for this static utility.
type JavaArrays struct{ _ byte }

func init()                                   { RegisterJavaType("java.util.Arrays", ObjectTypeID) }
func (*JavaArrays) JavaDynamicTypeID() TypeID { return "java.util.Arrays" }
