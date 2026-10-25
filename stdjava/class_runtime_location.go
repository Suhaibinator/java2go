package stdjava

import "sync"

// ClassLoaderIdentity identifies a fixed domain in the supported JDK21
// classpath profile. TypeID does not model classes from custom class loaders.
type ClassLoaderIdentity string

const (
	BootstrapClassLoaderIdentity ClassLoaderIdentity = "jdk21-classpath:bootstrap"
	PlatformClassLoaderIdentity  ClassLoaderIdentity = "jdk21-classpath:platform"
	AppClassLoaderIdentity       ClassLoaderIdentity = "jdk21-classpath:app"
)

// ClassRuntimeLocation is explicit declaration provenance. Its zero value is
// unknown; an empty ModuleName means unnamed only when Known is true. Values
// are copied on publication and lookup and never initialize a Java class.
type ClassRuntimeLocation struct {
	Known          bool
	ModuleName     string
	LoaderIdentity ClassLoaderIdentity
	LoaderName     string
}

var classRuntimeLocationRegistry sync.Map

// RegisterClassRuntimeLocation publishes one immutable classpath-profile fact.
// Compatible repeated publication is allowed; conflicting or incomplete facts
// are rejected. Array locations always derive from their ultimate component.
func RegisterClassRuntimeLocation(id TypeID, location ClassRuntimeLocation) {
	if !location.Known {
		if location != (ClassRuntimeLocation{}) {
			panic("unknown class runtime location contains facts")
		}
		return
	}
	if id == "" {
		panic("class runtime location requires a declaration")
	}
	if _, array := arrayComponentTypeID(id); array {
		panic("array runtime location derives from its component")
	}
	loaderName := ""
	switch location.LoaderIdentity {
	case BootstrapClassLoaderIdentity:
		loaderName = "bootstrap"
	case PlatformClassLoaderIdentity:
		loaderName = "platform"
	case AppClassLoaderIdentity:
		loaderName = "app"
	default:
		panic("unsupported class loader identity")
	}
	if location.LoaderName != loaderName {
		panic("class runtime location has inconsistent loader name")
	}
	if location.LoaderIdentity == AppClassLoaderIdentity && location.ModuleName != "" && !reflectionNamedApplicationLocation(id, location) {
		panic("named application modules are unsupported")
	}
	if location.LoaderIdentity != AppClassLoaderIdentity && location.ModuleName == "" {
		panic("runtime module provenance is incomplete")
	}
	previous, loaded := classRuntimeLocationRegistry.LoadOrStore(id, location)
	if loaded && previous.(ClassRuntimeLocation) != location {
		panic("conflicting class runtime location for " + string(id))
	}
}

// ClassRuntimeLocationOf returns only published declaration facts. Arrays peel
// to the ultimate component; primitive provenance is explicitly registered.
// Neither Java package spelling nor source ownership implies a location.
func ClassRuntimeLocationOf(id TypeID) ClassRuntimeLocation {
	for {
		component, array := arrayComponentTypeID(id)
		if !array {
			break
		}
		id = component
	}
	if value, known := classRuntimeLocationRegistry.Load(id); known {
		return value.(ClassRuntimeLocation)
	}
	return ClassRuntimeLocation{}
}
