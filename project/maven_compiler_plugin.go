package project

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// CompilerMetadata records the bounded compiler-plugin configuration recognized
// by source discovery. Discovery does not execute Maven or annotation processors.
// Consumers compiling Java must honor these values; Proc empty means unspecified,
// not an implicit request to disable annotation processing.
type CompilerMetadata struct {
	PluginVersion string
	Release       string
	Encoding      string
	Proc          string
}

// Preserve every field and attribute so unsupported plugin behavior cannot be
// silently discarded by encoding/xml's usual unknown-field handling.
type compilerPluginElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr              `xml:",any,attr"`
	Text     string                  `xml:",chardata"`
	Children []compilerPluginElement `xml:",any"`
}

func compilerElementFields(element compilerPluginElement, allowed map[string]bool) (map[string]compilerPluginElement, error) {
	if len(element.Attrs) != 0 || strings.TrimSpace(element.Text) != "" {
		return nil, fmt.Errorf("compiler plugin %s contains unsupported attributes or text", element.XMLName.Local)
	}
	fields := map[string]compilerPluginElement{}
	for _, child := range element.Children {
		name := child.XMLName.Local
		if child.XMLName.Space != "" && child.XMLName.Space != "http://maven.apache.org/POM/4.0.0" {
			return nil, fmt.Errorf("compiler plugin field %s has unsupported XML namespace", name)
		}
		if !allowed[name] {
			return nil, fmt.Errorf("compiler plugin field %s is unsupported", name)
		}
		if _, exists := fields[name]; exists {
			return nil, fmt.Errorf("compiler plugin field %s is duplicated", name)
		}
		fields[name] = child
	}
	return fields, nil
}

func compilerScalar(element compilerPluginElement, properties map[string]string) (string, error) {
	if len(element.Attrs) != 0 || len(element.Children) != 0 {
		return "", fmt.Errorf("compiler plugin %s must be a scalar without attributes", element.XMLName.Local)
	}
	return expand(element.Text, properties)
}

func resolveCompilerMetadata(plugins []compilerPluginElement, rawProperties []property, properties map[string]string) (*CompilerMetadata, error) {
	if len(plugins) == 0 {
		return nil, nil
	}
	if len(plugins) != 1 {
		return nil, fmt.Errorf("build plugins support one direct compiler plugin declaration only")
	}
	fields, err := compilerElementFields(plugins[0], map[string]bool{"groupId": true, "artifactId": true, "version": true, "configuration": true})
	if err != nil {
		return nil, fmt.Errorf("unsupported build plugins: %w", err)
	}
	coordinate := map[string]string{}
	for _, name := range []string{"groupId", "artifactId", "version"} {
		value, err := compilerScalar(fields[name], properties)
		if err != nil {
			return nil, err
		}
		coordinate[name] = value
	}
	if coordinate["groupId"] == "" {
		coordinate["groupId"] = "org.apache.maven.plugins"
	}
	if coordinate["groupId"] != "org.apache.maven.plugins" || coordinate["artifactId"] != "maven-compiler-plugin" {
		return nil, fmt.Errorf("unsupported build plugins: %s:%s", coordinate["groupId"], coordinate["artifactId"])
	}
	// The local official 3.16.0 plugin descriptor defines the supported property
	// defaults below. Other versions need their own reviewed metadata contract.
	if coordinate["version"] != "3.16.0" {
		return nil, fmt.Errorf("compiler plugin version %q is unsupported; metadata is validated for 3.16.0 only", coordinate["version"])
	}
	allowedProperties := map[string]bool{"maven.compiler.release": true, "maven.compiler.proc": true, "encoding": true, "project.build.sourceEncoding": true}
	for name := range properties {
		if strings.HasPrefix(name, "maven.compiler.") && !allowedProperties[name] {
			return nil, fmt.Errorf("compiler property %s is unsupported", name)
		}
	}
	seen := map[string]bool{}
	for _, property := range rawProperties {
		name := property.XMLName.Local
		if allowedProperties[name] {
			if seen[name] {
				return nil, fmt.Errorf("compiler property %s is duplicated", name)
			}
			seen[name] = true
		}
	}
	config := map[string]compilerPluginElement{}
	if element, exists := fields["configuration"]; exists {
		config, err = compilerElementFields(element, map[string]bool{"release": true, "encoding": true, "proc": true})
		if err != nil {
			return nil, err
		}
	}
	effective := func(name, propertyName, fallback string) (string, error) {
		if element, exists := config[name]; exists {
			return compilerScalar(element, properties)
		}
		value, exists := properties[propertyName]
		if !exists {
			value = properties[fallback]
		}
		return expand(value, properties)
	}
	metadata := &CompilerMetadata{PluginVersion: coordinate["version"]}
	metadata.Release, err = effective("release", "maven.compiler.release", "")
	if err != nil {
		return nil, err
	}
	metadata.Encoding, err = effective("encoding", "encoding", "project.build.sourceEncoding")
	if err != nil {
		return nil, err
	}
	metadata.Proc, err = effective("proc", "maven.compiler.proc", "")
	if err != nil {
		return nil, err
	}
	if metadata.Release != "21" {
		return nil, fmt.Errorf("compiler release %q is unsupported; source discovery requires explicit Java 21", metadata.Release)
	}
	if !strings.EqualFold(metadata.Encoding, "UTF-8") && !strings.EqualFold(metadata.Encoding, "UTF8") {
		return nil, fmt.Errorf("compiler encoding %q is unsupported; UTF-8 is required", metadata.Encoding)
	}
	metadata.Encoding = "UTF-8"
	if metadata.Proc != "" && metadata.Proc != "none" {
		return nil, fmt.Errorf("compiler proc %q is unsupported; explicit processing mode must be none", metadata.Proc)
	}
	return metadata, nil
}
