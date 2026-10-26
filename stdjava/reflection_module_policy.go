package stdjava

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// ReflectionModuleDescriptor is immutable runtime provenance, not an instruction
// to open a module. Named application source remains rejected by the frontend.
// A future producer must publish its complete owned package/access facts before
// attaching a class location to the module.
type ReflectionModuleDescriptor struct {
	Name           string
	LoaderIdentity ClassLoaderIdentity
	Packages       []string
	Exports        map[string][]string // present with an empty target list: unqualified
	Opens          map[string][]string
	Open           bool
}

var reflectionModules sync.Map

func RegisterReflectionModule(descriptor ReflectionModuleDescriptor) {
	if descriptor.Name == "" || descriptor.LoaderIdentity == "" {
		panic("reflection module provenance is incomplete")
	}
	switch descriptor.LoaderIdentity {
	case BootstrapClassLoaderIdentity, PlatformClassLoaderIdentity, AppClassLoaderIdentity:
	default:
		panic("unsupported reflection module loader")
	}
	packages := map[string]bool{}
	for _, name := range descriptor.Packages {
		if name == "" || packages[name] {
			panic("invalid reflection module package")
		}
		packages[name] = true
	}
	clone := func(input map[string][]string) map[string][]string {
		result := map[string][]string{}
		for name, targets := range input {
			if !packages[name] {
				panic("module access edge has no owned package")
			}
			result[name] = append([]string{}, targets...)
		}
		return result
	}
	descriptor.Packages = append([]string(nil), descriptor.Packages...)
	descriptor.Exports = clone(descriptor.Exports)
	descriptor.Opens = clone(descriptor.Opens)
	if old, loaded := reflectionModules.LoadOrStore(descriptor.Name, descriptor); loaded && !reflect.DeepEqual(old, descriptor) {
		panic("conflicting reflection module provenance")
	}
}
func reflectionModule(location ClassRuntimeLocation) ReflectionModuleDescriptor {
	if !location.Known {
		panic(NewUnsupportedOperationException("reflection module provenance is unavailable"))
	}
	if location.ModuleName == "" {
		return ReflectionModuleDescriptor{LoaderIdentity: location.LoaderIdentity, Open: true}
	}
	value, known := reflectionModules.Load(location.ModuleName)
	if !known {
		panic(NewUnsupportedOperationException("reflection named module policy is unavailable"))
	}
	module := value.(ReflectionModuleDescriptor)
	if module.LoaderIdentity != location.LoaderIdentity {
		panic(NewUnsupportedOperationException("reflection module loader provenance disagrees"))
	}
	return module
}
func reflectionNamedApplicationLocation(id TypeID, location ClassRuntimeLocation) bool {
	value, known := reflectionModules.Load(location.ModuleName)
	if !known {
		return false
	}
	module := value.(ReflectionModuleDescriptor)
	if module.LoaderIdentity != location.LoaderIdentity {
		return false
	}
	packageName := reflectionPackage(id)
	for _, name := range module.Packages {
		if name == packageName {
			return true
		}
	}
	return false
}
func reflectionPackage(id TypeID) string {
	name := string(id)
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		return name[:dot]
	}
	return ""
}
func reflectionSameModule(left, right ClassRuntimeLocation) bool {
	return left.Known && right.Known && left.ModuleName == right.ModuleName && left.LoaderIdentity == right.LoaderIdentity
}
func reflectionModuleEdge(module ReflectionModuleDescriptor, packageName string, caller ClassRuntimeLocation, open bool) bool {
	if module.Name == "" || module.Open {
		return true
	}
	allows := func(edges map[string][]string) bool {
		targets, present := edges[packageName]
		if !present {
			return false
		}
		if len(targets) == 0 {
			return true
		}
		for _, target := range targets {
			if target == caller.ModuleName && target != "" || target == "ALL-UNNAMED" && caller.ModuleName == "" {
				return true
			}
		}
		return false
	}
	// An open package is also exported at run time (Module.isExported).
	return allows(module.Opens) || !open && allows(module.Exports)
}

var reflectionCallerNests sync.Map

// Registration is emitted only for an actually retained source caller. It is a
// separate immutable nest fact for late classes, whose regular method metadata
// is intentionally bounded. It never changes class/module identity.
func RegisterReflectionCallerNest(caller, nest TypeID) {
	if caller == "" || nest == "" {
		panic("reflection caller nest provenance is incomplete")
	}
	if previous, loaded := reflectionCallerNests.LoadOrStore(caller, nest); loaded && previous.(TypeID) != nest {
		panic("conflicting reflection caller nest")
	}
}
func reflectionNest(id TypeID) TypeID {
	if nested, known := reflectionCallerNests.Load(id); known {
		return nested.(TypeID)
	}
	if host := classDescriptor(id).NestHost; host != "" {
		return host
	}
	return id
}
func reflectionCallerAccess(caller TypeID, member JavaAccessibleObject, receiver any) bool {
	declaring := member.GetDeclaringClass()
	owner := declaring.TypeID()
	modifiers := member.GetModifiers()
	if caller == "" {
		return modifiers&1 != 0 && declaring.GetModifiers()&1 != 0
	}
	callerLocation, _ := reflectionClassModule(caller)
	ownerLocation, module := reflectionClassModule(owner)
	sameModule := reflectionSameModule(callerLocation, ownerLocation)
	if !sameModule && !reflectionModuleEdge(module, reflectionPackage(owner), callerLocation, false) {
		return false
	}
	if caller == owner {
		return true
	}
	samePackage := callerLocation.LoaderIdentity == ownerLocation.LoaderIdentity && reflectionPackage(caller) == reflectionPackage(owner)
	if declaring.GetModifiers()&1 == 0 && !samePackage {
		return false
	}
	if modifiers&1 != 0 {
		return true
	}
	if modifiers&2 != 0 {
		return reflectionNest(caller) == reflectionNest(owner)
	}
	if samePackage {
		return true
	}
	if modifiers&4 == 0 || !JavaTypeAssignable(caller, owner) {
		return false
	}
	if modifiers&8 != 0 {
		return true
	}
	if _, constructor := member.(*Constructor); constructor {
		// Constructor.canAccess/newInstance use the declaring class as the
		// protected target class, not an instance of the accessing subclass.
		return JavaTypeAssignable(owner, caller)
	}
	dynamic, known := ObjectDynamicType(receiver)
	return known && JavaTypeAssignable(dynamic, caller)
}

type reflectionUnnamedModule struct{ loader ClassLoaderIdentity }

var reflectionUnnamedModules sync.Map

func reflectionUnnamedModuleText(loader ClassLoaderIdentity) string {
	value, _ := reflectionUnnamedModules.LoadOrStore(loader, &reflectionUnnamedModule{loader: loader})
	return "unnamed module @" + strconv.FormatUint(uint64(uint32(SystemIdentityHashCode(value))), 16)
}
func reflectionClassModule(id TypeID) (ClassRuntimeLocation, ReflectionModuleDescriptor) {
	location := ClassRuntimeLocationOf(id)
	module := reflectionModule(location)
	if module.Name != "" {
		owned := false
		for _, name := range module.Packages {
			if name == reflectionPackage(id) {
				owned = true
				break
			}
		}
		if !owned {
			panic(NewUnsupportedOperationException("class package has no owned module provenance"))
		}
	}
	return location, module
}
