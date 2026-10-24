package stdjava

import "golang.org/x/text/unicode/norm"

type NormalizerForm struct{ form norm.Form }

var (
	NormalizerNFC  = &NormalizerForm{norm.NFC}
	NormalizerNFD  = &NormalizerForm{norm.NFD}
	NormalizerNFKC = &NormalizerForm{norm.NFKC}
	NormalizerNFKD = &NormalizerForm{norm.NFKD}
)

func NormalizerNormalize(execution *Execution, value any, form *NormalizerForm) string {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(form)
	return form.form.String(StringValueOfExecution(execution, value))
}
func NormalizerIsNormalized(execution *Execution, value any, form *NormalizerForm) bool {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(form)
	return form.form.IsNormalString(StringValueOfExecution(execution, value))
}
func (*NormalizerForm) JavaDynamicTypeID() TypeID { return "java.text.Normalizer$Form" }
func init()                                       { RegisterJavaType("java.text.Normalizer$Form", ObjectTypeID) }
