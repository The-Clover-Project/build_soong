package common

import "encoding/json"

type Ramdisk16kImgPropertiesJSON struct {
	// List or filegroup of prebuilt kernel module files. Should have .ko suffix.
	Srcs []string `json:",omitempty"`

	// List or filegroup of prebuilt kernel module files that the debug symbols will be stripped.
	// Should have .ko suffix. These entries must be listed in srcs as well, otherwise an error
	// will be thrown. This is because ther order of the srcs is used for generating the
	// modules.load file if load property is not specified.
	Strip_symbol_srcs []string `json:",omitempty"`

	// List specifying load order of kernel modules.
	Load []string `json:",omitempty"`

	// Path to the prebuilt 16KB kernel
	Kernel *string `json:",omitempty"`
}

func (p *Ramdisk16kImgPropertiesJSON) ToJSON() string {
	result, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(result)
}
