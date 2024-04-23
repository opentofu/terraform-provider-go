package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Function struct {
	tfprotov6.Function
	Impl func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError)
}

type FunctionProvider struct {
	ProviderSchema   *tfprotov6.Schema
	StaticFunctions  map[string]*Function
	dynamicFunctions map[string]*Function
	Configure        func(*tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic)
}

func (f *FunctionProvider) GetMetadata(context.Context, *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	var functions []tfprotov6.FunctionMetadata
	for name := range f.StaticFunctions {
		functions = append(functions, tfprotov6.FunctionMetadata{Name: name})
	}

	return &tfprotov6.GetMetadataResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) GetProviderSchema(context.Context, *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetProviderSchemaResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Provider:           f.ProviderSchema,
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) ValidateProviderConfig(ctx context.Context, req *tfprotov6.ValidateProviderConfigRequest) (*tfprotov6.ValidateProviderConfigResponse, error) {
	// Passthrough
	return &tfprotov6.ValidateProviderConfigResponse{PreparedConfig: req.Config}, nil
}
func (f *FunctionProvider) ConfigureProvider(ctx context.Context, req *tfprotov6.ConfigureProviderRequest) (*tfprotov6.ConfigureProviderResponse, error) {
	funcs, diags := f.Configure(req.Config)
	f.dynamicFunctions = funcs
	return &tfprotov6.ConfigureProviderResponse{
		Diagnostics: diags,
	}, nil
}
func (f *FunctionProvider) StopProvider(context.Context, *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	return &tfprotov6.StopProviderResponse{}, nil
}
func (f *FunctionProvider) ValidateResourceConfig(context.Context, *tfprotov6.ValidateResourceConfigRequest) (*tfprotov6.ValidateResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) UpgradeResourceState(context.Context, *tfprotov6.UpgradeResourceStateRequest) (*tfprotov6.UpgradeResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadResource(context.Context, *tfprotov6.ReadResourceRequest) (*tfprotov6.ReadResourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) PlanResourceChange(context.Context, *tfprotov6.PlanResourceChangeRequest) (*tfprotov6.PlanResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ApplyResourceChange(context.Context, *tfprotov6.ApplyResourceChangeRequest) (*tfprotov6.ApplyResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ImportResourceState(context.Context, *tfprotov6.ImportResourceStateRequest) (*tfprotov6.ImportResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ValidateDataResourceConfig(context.Context, *tfprotov6.ValidateDataResourceConfigRequest) (*tfprotov6.ValidateDataResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadDataSource(context.Context, *tfprotov6.ReadDataSourceRequest) (*tfprotov6.ReadDataSourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) CallFunction(ctx context.Context, req *tfprotov6.CallFunctionRequest) (*tfprotov6.CallFunctionResponse, error) {
	if fn, ok := f.StaticFunctions[req.Name]; ok {
		ret, err := fn.Impl(req.Arguments)
		return &tfprotov6.CallFunctionResponse{
			Result: ret,
			Error:  err,
		}, nil
	}
	if f.dynamicFunctions != nil {
		if fn, ok := f.dynamicFunctions[req.Name]; ok {
			ret, err := fn.Impl(req.Arguments)
			return &tfprotov6.CallFunctionResponse{
				Result: ret,
				Error:  err,
			}, nil
		}
	}
	return nil, errors.New("unknown function " + req.Name)
}
func (f *FunctionProvider) GetFunctions(context.Context, *tfprotov6.GetFunctionsRequest) (*tfprotov6.GetFunctionsResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}
	for name, fn := range f.dynamicFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetFunctionsResponse{
		Functions: functions,
	}, nil
}

func main() {
	err := tf6server.Serve("registry.opentofu.org/opentofu/go", func() tfprotov6.ProviderServer {
		provider := &FunctionProvider{
			ProviderSchema: &tfprotov6.Schema{
				Block: &tfprotov6.SchemaBlock{
					Attributes: []*tfprotov6.SchemaAttribute{
						&tfprotov6.SchemaAttribute{
							Name:     "go",
							Type:     tftypes.String,
							Required: true,
						},
					},
				},
			},
			Configure: func(config *tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic) {
				res, err := config.Unmarshal(tftypes.Map{ElementType: tftypes.String})
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}
				cfg := make(map[string]tftypes.Value)
				err = res.As(&cfg)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				codeVal := cfg["go"]
				var code string
				err = codeVal.As(&code)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				interpreter := interp.New(interp.Options{})
				if err := interpreter.Use(stdlib.Symbols); err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Failed to load Go standard library",
						Detail:   err.Error(),
					}}
				}

				_, err = interpreter.Eval(code)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Failed to evaluate Go code",
						Detail:   err.Error(),
					}}
				}

				exports := interpreter.Symbols("lib")
				libExports := exports["lib"]

				functions := map[string]*Function{}
				for name, export := range libExports {
					if export.Kind() != reflect.Func {
						continue
					}
					fn, diags := GoFunctionToTFFunction(interpreter, export)
					if len(diags) > 0 {
						return nil, diags
					}
					functions[GoNameToTFName(name)] = fn
				}

				return functions, nil
			},
			StaticFunctions: map[string]*Function{},
		}
		return provider
	})
	if err != nil {
		panic(err)
	}
}

func GoFunctionToTFFunction(interpreter *interp.Interpreter, fn reflect.Value) (*Function, []*tfprotov6.Diagnostic) {
	exportType := fn.Type()
	var parameters []*tfprotov6.FunctionParameter
	for i := 0; i < exportType.NumIn(); i++ {
		functionParameter, err := GoTypeToTFFunctionParam(exportType.In(i))
		if err != nil {
			return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
				Severity: tfprotov6.DiagnosticSeverityError,
				Summary:  "Failed to convert Argument type to TF type",
				Detail:   fmt.Errorf("argument %d: %w", i, err).Error(),
			}}
		}
		parameters = append(parameters, functionParameter)
	}
	if exportType.NumOut() == 0 {
		return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Function must return a value",
		}}
	}
	if exportType.NumOut() > 2 {
		return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Function must return at most two values",
		}}
	}
	if exportType.NumOut() == 2 && exportType.Out(1) != reflect.TypeFor[error]() {
		return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Second return value, if exists, must be an error",
		}}
	}
	output := exportType.Out(0)
	outputType, err := GoTypeToTFType(output)
	if err != nil {
		return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Failed to convert Function output type to TF type",
			Detail:   err.Error(),
		}}
	}
	return &Function{
		Function: tfprotov6.Function{
			Parameters: parameters,
			Return: &tfprotov6.FunctionReturn{
				Type: outputType,
			},
		},
		Impl: func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError) {
			goArgs := make([]reflect.Value, len(args))
			for i, arg := range args {
				var err error
				goArg, err := ProtoToGo(parameters[i].Type, exportType.In(i), arg)
				if err != nil {
					return nil, &tfprotov6.FunctionError{
						Text: err.Error(),
					}
				}
				goArgs[i] = reflect.ValueOf(goArg)
			}
			goResult := fn.Call(goArgs)
			if len(goResult) > 1 && !goResult[1].IsNil() {
				err := goResult[1].Interface().(error)
				if err != nil {
					return nil, &tfprotov6.FunctionError{
						Text: err.Error(),
					}
				}
			}

			out, err := GoToProto(outputType, goResult[0].Interface())
			if err != nil {
				return nil, &tfprotov6.FunctionError{
					Text: err.Error(),
				}
			}
			return out, nil
		},
	}, nil
}

func TfValueToProto(tfType tftypes.Type, tfVal tftypes.Value) (*tfprotov6.DynamicValue, error) {
	value, err := tfprotov6.NewDynamicValue(tfType, tfVal)
	return &value, err
}

func GoTypeToTFFunctionParam(t reflect.Type) (*tfprotov6.FunctionParameter, error) {
	outType, err := GoTypeToTFType(t)
	if err != nil {
		return nil, err
	}

	return &tfprotov6.FunctionParameter{
		AllowUnknownValues: false,
		AllowNullValue:     t.Kind() == reflect.Ptr,
		Type:               outType,
	}, nil
}

func GoTypeToTFType(t reflect.Type) (tftypes.Type, error) {
	switch t.Kind() {
	case reflect.String:
		return tftypes.String, nil
	case reflect.Bool:
		return tftypes.Bool, nil
	case reflect.Int, reflect.Float64:
		return tftypes.Number, nil
	case reflect.Ptr:
		return GoTypeToTFType(t.Elem())
	case reflect.Interface:
		if reflect.TypeFor[interface{}]().Implements(t) {
			return tftypes.DynamicPseudoType, nil
		} else {
			return nil, fmt.Errorf("unsupported interface type %s, only interface{}/any interface type is supported", t.String())
		}
	case reflect.Slice:
		elementType, err := GoTypeToTFType(t.Elem())
		if err != nil {
			return nil, err
		}
		return tftypes.List{
			ElementType: elementType,
		}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type %s, only string keys are supported", t.Key().String())
		}
		valueType, err := GoTypeToTFType(t.Elem())
		if err != nil {
			return nil, err
		}
		return tftypes.Map{
			ElementType: valueType,
		}, nil
	case reflect.Struct:
		attributeTypes := make(map[string]tftypes.Type)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldType, err := GoTypeToTFType(field.Type)
			if err != nil {
				return nil, err
			}
			attributeTypes[getTfObjectGoFieldName(field)] = fieldType
		}
		return tftypes.Object{
			AttributeTypes: attributeTypes,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", t.String())
	}
}

func getTfObjectGoFieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("tf"); tag != "" {
		return tag
	}
	return uncapitalize(field.Name)
}

func uncapitalize(s string) string {
	if len(s) == 1 {
		return strings.ToLower(s)
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func GoNameToTFName(name string) string {
	return strings.ToLower(name)
}

func ProtoToGo(argumentTfType tftypes.Type, argumentGoType reflect.Type, arg *tfprotov6.DynamicValue) (any, error) {
	if len(arg.JSON) == 0 && len(arg.MsgPack) == 0 {
		// This is an edge-case not properly handled by arg.IsNull().
		// It happens when you pass (from tf) the value `null`, to a function expecting e.g. a string pointer.

		// We can't just return nil here, because we need a *typed* interface{} :)
		// If we'd return nil here, then the later reflect call of our dynamically created functions
		// would fail during the dynamic type-check.
		return reflect.Zero(argumentGoType).Interface(), nil
	}
	argTf, err := arg.Unmarshal(argumentTfType)
	if err != nil {
		return nil, err
	}

	return TfToGoValue(argumentGoType, argTf)
}

func TfToGoValue(goType reflect.Type, tfValue tftypes.Value) (any, error) {
	if tfValue.IsNull() {
		return nil, nil
	}

	switch goType.Kind() {
	case reflect.String:
		var str string
		if err := tfValue.As(&str); err != nil {
			return nil, err
		}
		return str, nil
	case reflect.Bool:
		var b bool
		if err := tfValue.As(&b); err != nil {
			return nil, err
		}
		return b, nil
	case reflect.Int:
		var bigFloat big.Float
		if err := tfValue.As(&bigFloat); err != nil {
			return nil, err
		}

		f, _ := bigFloat.Int64()
		return int(f), nil
	case reflect.Float64:
		var bigFloat big.Float
		if err := tfValue.As(&bigFloat); err != nil {
			return nil, err
		}

		f, _ := bigFloat.Int64()
		return f, nil
	case reflect.Ptr:
		if tfValue.IsNull() {
			return nil, nil
		}
		value, err := TfToGoValue(goType.Elem(), tfValue)
		if err != nil {
			return nil, err
		}
		// If we return &value, then the type will be *interface{}.
		// So we construct a concrete type pointer via reflect.
		// This way, we get e.g. *string instead of *interface{}.
		out := reflect.New(reflect.TypeOf(value))
		out.Elem().Set(reflect.ValueOf(value))
		return out.Interface(), nil
	case reflect.Interface:
		panic("implement interface{}")
	case reflect.Slice:
		var tfValues []tftypes.Value
		if err := tfValue.As(&tfValues); err != nil {
			return nil, err
		}

		out := reflect.MakeSlice(goType, len(tfValues), len(tfValues))
		for i := 0; i < len(tfValues); i++ {
			elem, err := TfToGoValue(goType.Elem(), tfValues[i])
			if err != nil {
				return nil, err
			}
			out.Index(i).Set(reflect.ValueOf(elem))
		}
		return out.Interface(), nil
	case reflect.Map:
		var tfMap map[string]tftypes.Value
		if err := tfValue.As(&tfMap); err != nil {
			return nil, err
		}
		out := reflect.MakeMap(goType)
		for key, tfElement := range tfMap {
			elem, err := TfToGoValue(goType.Elem(), tfElement)
			if err != nil {
				return nil, err
			}
			out.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem))
		}
		return out.Interface(), nil
	case reflect.Struct:
		var tfMap map[string]tftypes.Value
		if err := tfValue.As(&tfMap); err != nil {
			return nil, err
		}
		// This is a fun one, you'd fine reflect.Zero(goType) should do the same, right?
		// Nope! If you use reflect.Zero, then the fields of it won't be addressable.
		// If the fields aren't addressable, they're not settable.
		// So, we use reflect.New and then take the pointed-to value, this way it is in fact addressable.
		out := reflect.New(goType).Elem()
		for i := 0; i < goType.NumField(); i++ {
			field := goType.Field(i)
			tfName := getTfObjectGoFieldName(field)
			tfElement, ok := tfMap[tfName]
			if !ok {
				return nil, fmt.Errorf("missing object field %s", tfName)
			}
			elem, err := TfToGoValue(field.Type, tfElement)
			if err != nil {
				return nil, err
			}
			out.Field(i).Set(reflect.ValueOf(elem))
		}
		return out.Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported type %s", goType.String())
	}
}

// func CtyToGo(goType reflect.Type, ctyValue cty.Value) (any, error) {
// 	ctyType := ctyValue.Type()
// 	switch goType.Kind() {
// 	case reflect.String:
// 		if ctyType != cty.String {
// 			return nil, fmt.Errorf("expected string, got %s", ctyType.FriendlyName())
// 		}
// 		return ctyValue.AsString(), nil
// 	case reflect.Bool:
// 		if ctyType != cty.Bool {
// 			return nil, fmt.Errorf("expected bool, got %s", ctyType.FriendlyName())
// 		}
// 		return ctyValue.True(), nil
// 	case reflect.Int:
// 		if ctyType != cty.Number {
// 			return nil, fmt.Errorf("expected number, got %s", ctyType.FriendlyName())
// 		}
// 		f, _ := ctyValue.AsBigFloat().Int64()
// 		return int(f), nil
// 	case reflect.Float64:
// 		if ctyType != cty.Number {
// 			return nil, fmt.Errorf("expected number, got %s", ctyType.FriendlyName())
// 		}
// 		f, _ := ctyValue.AsBigFloat().Float64()
// 		return f, nil
// 	case reflect.Ptr:
// 		if ctyValue.IsNull() {
// 			return nil, nil
// 		}
// 		return CtyToGo(goType.Elem(), ctyValue)
// 	case reflect.Interface:
// 		panic("implement interface{}")
// 	case reflect.Slice:
// 		if !ctyType.IsListType() {
// 			return nil, fmt.Errorf("expected list, got %s", ctyType.FriendlyName())
// 		}
// 		out := reflect.MakeSlice(goType, ctyValue.LengthInt(), ctyValue.LengthInt())
// 		for i := 0; i < ctyValue.LengthInt(); i++ {
// 			elem, err := CtyToGo(goType.Elem(), ctyValue.Index(cty.NumberIntVal(int64(i))))
// 			if err != nil {
// 				return nil, err
// 			}
// 			out.Index(i).Set(reflect.ValueOf(elem))
// 		}
// 		return out.Interface(), nil
// 	case reflect.Map:
// 		if !ctyType.IsMapType() {
// 			return nil, fmt.Errorf("expected map, got %s", ctyType.FriendlyName())
// 		}
// 		out := reflect.MakeMap(goType)
// 		for key, ctyElement := range ctyValue.AsValueMap() {
// 			elem, err := CtyToGo(goType.Elem(), ctyElement)
// 			if err != nil {
// 				return nil, err
// 			}
// 			out.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem))
// 		}
// 		return out.Interface(), nil
// 	default:
// 		return nil, fmt.Errorf("unsupported type %s", goType.String())
// 	}
// }

func GoToProto(tfType tftypes.Type, value any) (*tfprotov6.DynamicValue, error) {
	tfValue, err := GoToTfValue(tfType, value)
	if err != nil {
		return nil, err
	}
	return TfValueToProto(tfType, tfValue)
}

func GoToTfValue(tfType tftypes.Type, value any) (tftypes.Value, error) {
	if value == nil {
		if err := tftypes.ValidateValue(tfType, nil); err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tfType, nil), nil
	}

	switch {
	case tfType.Is(tftypes.String):
		return tftypes.NewValue(tftypes.String, value), nil
	case tfType.Is(tftypes.Bool):
		return tftypes.NewValue(tftypes.Bool, value), nil
	case tfType.Is(tftypes.Number):
		switch value := value.(type) {
		case int:
			return tftypes.NewValue(tftypes.Number, value), nil
		case float64:
			return tftypes.NewValue(tftypes.Number, value), nil
		default:
			return tftypes.Value{}, fmt.Errorf("expected number, got %T", value)
		}
	case tfType.Is(tftypes.DynamicPseudoType):
		panic("implement interface{}")
	default:
		switch tfType := tfType.(type) {
		case tftypes.List:
			if reflect.TypeOf(value).Kind() != reflect.Slice {
				return tftypes.Value{}, fmt.Errorf("expected slice, got %T", value)
			}
			slice := reflect.ValueOf(value)
			out := make([]tftypes.Value, slice.Len())
			for i := 0; i < slice.Len(); i++ {
				elem, err := GoToTfValue(tfType.ElementType, slice.Index(i).Interface())
				if err != nil {
					return tftypes.Value{}, err
				}
				out[i] = elem
			}
			return tftypes.NewValue(tfType, out), nil
		case tftypes.Map:
			if reflect.TypeOf(value).Kind() != reflect.Map {
				return tftypes.Value{}, fmt.Errorf("expected map, got %T", value)
			}
			m := reflect.ValueOf(value)
			out := make(map[string]tftypes.Value, m.Len())
			for _, key := range m.MapKeys() {
				elem, err := GoToTfValue(tfType.ElementType, m.MapIndex(key).Interface())
				if err != nil {
					return tftypes.Value{}, err
				}
				out[key.String()] = elem
			}
			return tftypes.NewValue(tfType, out), nil
		case tftypes.Object:
			if reflect.TypeOf(value).Kind() != reflect.Struct {
				return tftypes.Value{}, fmt.Errorf("expected struct, got %T", value)
			}
			out := make(map[string]tftypes.Value, len(tfType.AttributeTypes))
			for i := 0; i < reflect.TypeOf(value).NumField(); i++ {
				field := reflect.TypeOf(value).Field(i)
				tfName := getTfObjectGoFieldName(field)
				elem, err := GoToTfValue(tfType.AttributeTypes[tfName], reflect.ValueOf(value).Field(i).Interface())
				if err != nil {
					return tftypes.Value{}, err
				}
				out[tfName] = elem
			}
			return tftypes.NewValue(tfType, out), nil
		default:
			return tftypes.Value{}, fmt.Errorf("unsupported type %s", tfType.String())
		}
	}
}
