// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// GraphQL schema generation per AI.md PART 14
// Schema is generated from code, not manually edited
package graphql

// Schema represents the GraphQL schema
type Schema struct {
	types      map[string]*TypeDef
	queryType  *TypeDef
	mutType    *TypeDef
}

// TypeDef represents a GraphQL type definition
type TypeDef struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind"`
	Description string      `json:"description,omitempty"`
	Fields      []*FieldDef `json:"fields,omitempty"`
	InputFields []*FieldDef `json:"inputFields,omitempty"`
	Args        []*FieldDef `json:"args,omitempty"`
}

// FieldDef represents a field definition
type FieldDef struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Type        *TypeRef `json:"type"`
	Args        []*Arg   `json:"args,omitempty"`
}

// TypeRef represents a type reference
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name,omitempty"`
	OfType *TypeRef `json:"ofType,omitempty"`
}

// Arg represents a field argument
type Arg struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Type         *TypeRef `json:"type"`
	DefaultValue string   `json:"defaultValue,omitempty"`
}

// NewSchema creates and returns the CasPb GraphQL schema
func NewSchema() *Schema {
	s := &Schema{
		types: make(map[string]*TypeDef),
	}
	s.buildSchema()
	return s
}

// buildSchema constructs the schema from code definitions
func (s *Schema) buildSchema() {
	// Scalar types
	s.types["String"] = &TypeDef{Name: "String", Kind: "SCALAR"}
	s.types["Int"] = &TypeDef{Name: "Int", Kind: "SCALAR"}
	s.types["Boolean"] = &TypeDef{Name: "Boolean", Kind: "SCALAR"}
	s.types["ID"] = &TypeDef{Name: "ID", Kind: "SCALAR"}

	// Health type
	s.types["Health"] = &TypeDef{
		Name:        "Health",
		Kind:        "OBJECT",
		Description: "Server health status",
		Fields: []*FieldDef{
			{Name: "ok", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "Boolean"}}},
			{Name: "status", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "version", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
		},
	}

	// ServerInfo type
	s.types["ServerInfo"] = &TypeDef{
		Name:        "ServerInfo",
		Kind:        "OBJECT",
		Description: "Server information and configuration",
		Fields: []*FieldDef{
			{Name: "version", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "title", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "public", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "maxBodyLen", Type: &TypeRef{Kind: "SCALAR", Name: "Int"}},
			{Name: "maxTitleLen", Type: &TypeRef{Kind: "SCALAR", Name: "Int"}},
			{Name: "lexers", Type: &TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "SCALAR", Name: "String"}}},
		},
	}

	// Paste type
	s.types["Paste"] = &TypeDef{
		Name:        "Paste",
		Kind:        "OBJECT",
		Description: "A paste entry",
		Fields: []*FieldDef{
			{Name: "id", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "ID"}}},
			{Name: "title", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "body", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "syntax", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "createTime", Type: &TypeRef{Kind: "SCALAR", Name: "Int"}},
			{Name: "deleteTime", Type: &TypeRef{Kind: "SCALAR", Name: "Int"}},
			{Name: "oneUse", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "isPrivate", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "isFile", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "fileName", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "mimeType", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "isUrl", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "originalUrl", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
		},
	}

	// PasteSummary type (for list)
	s.types["PasteSummary"] = &TypeDef{
		Name:        "PasteSummary",
		Kind:        "OBJECT",
		Description: "Summary of a paste for list view",
		Fields: []*FieldDef{
			{Name: "id", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "ID"}}},
			{Name: "title", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "syntax", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "createTime", Type: &TypeRef{Kind: "SCALAR", Name: "Int"}},
		},
	}

	// CreatePasteResult type
	s.types["CreatePasteResult"] = &TypeDef{
		Name:        "CreatePasteResult",
		Kind:        "OBJECT",
		Description: "Result of creating a paste",
		Fields: []*FieldDef{
			{Name: "id", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "ID"}}},
			{Name: "url", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
		},
	}

	// PasteInput type
	s.types["PasteInput"] = &TypeDef{
		Name:        "PasteInput",
		Kind:        "INPUT_OBJECT",
		Description: "Input for creating a paste",
		InputFields: []*FieldDef{
			{Name: "title", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "body", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "String"}}},
			{Name: "syntax", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "expiration", Type: &TypeRef{Kind: "SCALAR", Name: "String"}},
			{Name: "oneUse", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
			{Name: "isPrivate", Type: &TypeRef{Kind: "SCALAR", Name: "Boolean"}},
		},
	}

	// Query type
	s.queryType = &TypeDef{
		Name:        "Query",
		Kind:        "OBJECT",
		Description: "Root query type",
		Fields: []*FieldDef{
			{
				Name:        "healthz",
				Description: "Get server health status",
				Type:        &TypeRef{Kind: "OBJECT", Name: "Health"},
			},
			{
				Name:        "serverInfo",
				Description: "Get server information",
				Type:        &TypeRef{Kind: "OBJECT", Name: "ServerInfo"},
			},
			{
				Name:        "paste",
				Description: "Get a paste by ID",
				Type:        &TypeRef{Kind: "OBJECT", Name: "Paste"},
				Args: []*Arg{
					{Name: "id", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "SCALAR", Name: "ID"}}},
				},
			},
			{
				Name:        "pastes",
				Description: "List public pastes",
				Type:        &TypeRef{Kind: "LIST", OfType: &TypeRef{Kind: "OBJECT", Name: "PasteSummary"}},
			},
		},
	}
	s.types["Query"] = s.queryType

	// Mutation type
	s.mutType = &TypeDef{
		Name:        "Mutation",
		Kind:        "OBJECT",
		Description: "Root mutation type",
		Fields: []*FieldDef{
			{
				Name:        "createPaste",
				Description: "Create a new paste",
				Type:        &TypeRef{Kind: "OBJECT", Name: "CreatePasteResult"},
				Args: []*Arg{
					{Name: "input", Type: &TypeRef{Kind: "NON_NULL", OfType: &TypeRef{Kind: "INPUT_OBJECT", Name: "PasteInput"}}},
				},
			},
		},
	}
	s.types["Mutation"] = s.mutType
}

// Introspect returns the schema for introspection queries
func (s *Schema) Introspect() map[string]interface{} {
	types := make([]interface{}, 0, len(s.types))
	for _, t := range s.types {
		types = append(types, s.typeToIntrospection(t))
	}

	return map[string]interface{}{
		"queryType":        map[string]string{"name": "Query"},
		"mutationType":     map[string]string{"name": "Mutation"},
		"subscriptionType": nil,
		"types":            types,
		"directives":       []interface{}{},
	}
}

// typeToIntrospection converts a type to introspection format
func (s *Schema) typeToIntrospection(t *TypeDef) map[string]interface{} {
	result := map[string]interface{}{
		"name":        t.Name,
		"kind":        t.Kind,
		"description": t.Description,
	}

	if t.Fields != nil {
		fields := make([]interface{}, len(t.Fields))
		for i, f := range t.Fields {
			fields[i] = s.fieldToIntrospection(f)
		}
		result["fields"] = fields
	}

	if t.InputFields != nil {
		inputFields := make([]interface{}, len(t.InputFields))
		for i, f := range t.InputFields {
			inputFields[i] = s.fieldToIntrospection(f)
		}
		result["inputFields"] = inputFields
	}

	return result
}

// fieldToIntrospection converts a field to introspection format
func (s *Schema) fieldToIntrospection(f *FieldDef) map[string]interface{} {
	result := map[string]interface{}{
		"name":        f.Name,
		"description": f.Description,
		"type":        s.typeRefToIntrospection(f.Type),
	}

	if f.Args != nil {
		args := make([]interface{}, len(f.Args))
		for i, a := range f.Args {
			args[i] = map[string]interface{}{
				"name":         a.Name,
				"description":  a.Description,
				"type":         s.typeRefToIntrospection(a.Type),
				"defaultValue": a.DefaultValue,
			}
		}
		result["args"] = args
	} else {
		result["args"] = []interface{}{}
	}

	return result
}

// typeRefToIntrospection converts a type reference to introspection format
func (s *Schema) typeRefToIntrospection(t *TypeRef) map[string]interface{} {
	if t == nil {
		return nil
	}

	result := map[string]interface{}{
		"kind": t.Kind,
	}

	if t.Name != "" {
		result["name"] = t.Name
	}

	if t.OfType != nil {
		result["ofType"] = s.typeRefToIntrospection(t.OfType)
	}

	return result
}

// GetSDL returns the schema in SDL format
func (s *Schema) GetSDL() string {
	sdl := `# CasPb GraphQL Schema
# Generated from code - do not edit manually

type Query {
  healthz: Health
  serverInfo: ServerInfo
  paste(id: ID!): Paste
  pastes: [PasteSummary]
}

type Mutation {
  createPaste(input: PasteInput!): CreatePasteResult
}

type Health {
  ok: Boolean!
  status: String
  version: String
}

type ServerInfo {
  version: String
  title: String
  public: Boolean
  maxBodyLen: Int
  maxTitleLen: Int
  lexers: [String]
}

type Paste {
  id: ID!
  title: String
  body: String
  syntax: String
  createTime: Int
  deleteTime: Int
  oneUse: Boolean
  isPrivate: Boolean
  isFile: Boolean
  fileName: String
  mimeType: String
  isUrl: Boolean
  originalUrl: String
}

type PasteSummary {
  id: ID!
  title: String
  syntax: String
  createTime: Int
}

type CreatePasteResult {
  id: ID!
  url: String
}

input PasteInput {
  title: String
  body: String!
  syntax: String
  expiration: String
  oneUse: Boolean
  isPrivate: Boolean
}
`
	return sdl
}
