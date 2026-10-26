package stdjava

// Modifier flag queries use the public java.lang.reflect.Modifier masks.
// Unknown and VM-only flag bits do not affect a query for a public flag.
const (
	ModifierPublic       int32 = 1
	ModifierPrivate      int32 = 2
	ModifierProtected    int32 = 4
	ModifierStatic       int32 = 8
	ModifierFinal        int32 = 16
	ModifierSynchronized int32 = 32
	ModifierVolatile     int32 = 64
	ModifierTransient    int32 = 128
	ModifierNative       int32 = 256
	ModifierInterface    int32 = 512
	ModifierAbstract     int32 = 1024
	ModifierStrict       int32 = 2048
)

func ModifierIsPublic(modifiers int32) bool {
	return modifiers&ModifierPublic != 0
}

func ModifierIsPrivate(modifiers int32) bool {
	return modifiers&ModifierPrivate != 0
}

func ModifierIsProtected(modifiers int32) bool {
	return modifiers&ModifierProtected != 0
}

func ModifierIsStatic(modifiers int32) bool {
	return modifiers&ModifierStatic != 0
}

func ModifierIsFinal(modifiers int32) bool {
	return modifiers&ModifierFinal != 0
}

func ModifierIsSynchronized(modifiers int32) bool {
	return modifiers&ModifierSynchronized != 0
}

func ModifierIsVolatile(modifiers int32) bool {
	return modifiers&ModifierVolatile != 0
}

func ModifierIsTransient(modifiers int32) bool {
	return modifiers&ModifierTransient != 0
}

func ModifierIsNative(modifiers int32) bool {
	return modifiers&ModifierNative != 0
}

func ModifierIsInterface(modifiers int32) bool {
	return modifiers&ModifierInterface != 0
}

func ModifierIsAbstract(modifiers int32) bool {
	return modifiers&ModifierAbstract != 0
}

func ModifierIsStrict(modifiers int32) bool {
	return modifiers&ModifierStrict != 0
}
