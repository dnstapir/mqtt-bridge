package schemaval

import (
	"bytes"
	"errors"
	"strings"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

/* Default schema accepts everything */
const cDEFAULT_SCHEMA = "{}"

type Schemaval struct {
	log    shared.LoggerIF
	schema *jsonschema.Schema
}

type Conf struct {
	Log      shared.LoggerIF
	Filename string
}

func Create(conf Conf) (*Schemaval, error) {
	newSchemaval := new(Schemaval)

	if conf.Log == nil {
		return nil, errors.New("error setting logger")
	}
	newSchemaval.log = conf.Log

	var schema *jsonschema.Schema
	if conf.Filename == "" {
		var err error

		newSchemaval.log.Warning("No JSON schema configured, will not enforce schema!")

		defaultSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(cDEFAULT_SCHEMA))
		if err != nil {
			return nil, errors.New("error unmarshaling default schema")
		}

		c := jsonschema.NewCompiler()
		err = c.AddResource("./default.json", defaultSchema)
		if err != nil {
			return nil, errors.New("error adding default schema")
		}

		schema, err = c.Compile("./default.json")
		if err != nil {
			return nil, errors.New("error compiling default schema")
		}
	} else {
		var err error

		c := jsonschema.NewCompiler()

		schema, err = c.Compile(conf.Filename)
		if err != nil {
			return nil, err
		}
	}

	newSchemaval.schema = schema

	if newSchemaval.schema == nil {
		panic("schema creation fault")
	}

	return newSchemaval, nil
}

func (s *Schemaval) Validate(data []byte) bool {
	dataReader := bytes.NewReader(data)
	obj, err := jsonschema.UnmarshalJSON(dataReader)
	if err != nil {
		s.log.Error("Error unmarshalling byte stream into JSON object")
		return false
	}

	err = s.schema.Validate(obj)
	if err != nil {
		s.log.Debug("Validation error '%s'", err)
		return false
	}

	return true
}
