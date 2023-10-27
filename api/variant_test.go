package api

import (
	"testing"

	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/suite"
)

type VariantTestSuite struct {
	suite.Suite
	VariantWithOverride    []VariantInternal
	VariantWithoutOverride []VariantInternal
}

func (suite *VariantTestSuite) SetupTest() {
	suite.VariantWithOverride = []VariantInternal{
		VariantInternal{
			Variant: Variant{
				Name: "VarA",
				Payload: Payload{
					Type:  "string",
					Value: "Test 1",
				},
			},
			Weight: 33,
			Overrides: []Override{
				Override{
					ContextName: "userId",
					Values: []string{
						"1",
					},
				},
				Override{
					ContextName: "sessionId",
					Values: []string{
						"ABCDE",
					},
				},
			},
		},
		VariantInternal{
			Variant: Variant{
				Name: "VarB",
				Payload: Payload{
					Type:  "string",
					Value: "Test 2",
				},
			},
			Weight: 33,
			Overrides: []Override{
				Override{
					ContextName: "remoteAddress",
					Values: []string{
						"127.0.0.1",
					},
				},
			},
		},
		VariantInternal{
			Variant: Variant{
				Name: "VarC",
				Payload: Payload{
					Type:  "string",
					Value: "Test 3",
				},
			},
			Weight: 34,
			Overrides: []Override{
				Override{
					ContextName: "env",
					Values: []string{
						"dev",
					},
				},
			},
		},
	}

	suite.VariantWithoutOverride = []VariantInternal{
		{
			Variant: Variant{
				Name: "VarD",
			},
			Weight: 33,
		},
		{
			Variant: Variant{
				Name: "VarE",
			},
			Weight: 33,
		},
		{
			Variant: Variant{
				Name: "VarF",
			},
			Weight: 34,
		},
	}
}

func (suite *VariantTestSuite) TestGetVariantWhenFeatureHasNoVariant() {
	mockFeature := Feature{
		Name:    "test.variants",
		Enabled: true,
	}
	mockContext := &context.Context{}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)

	suite.Equal(DISABLED_VARIANT, variantSetup, "Should return default variant")
}

func (suite *VariantTestSuite) TestGetVariant_OverrideOnUserId() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithOverride,
	}
	mockContext := &context.Context{
		UserId:        "1",
		SessionId:     "ABCDE",
		RemoteAddress: "127.0.0.1",
	}
	expectedPayload := Payload{
		Type:  "string",
		Value: "Test 1",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarA", variantSetup.Name, "Should return VarA")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
	suite.Equal(expectedPayload, variantSetup.Payload, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_OverrideOnRemoteAddress() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithOverride,
	}
	mockContext := &context.Context{
		SessionId:     "FGHIJ",
		RemoteAddress: "127.0.0.1",
	}
	expectedPayload := Payload{
		Type:  "string",
		Value: "Test 2",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarB", variantSetup.Name, "Should return VarB")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
	suite.Equal(expectedPayload, variantSetup.Payload, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_OverrideOnSessionId() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithOverride,
	}
	mockContext := &context.Context{
		UserId:        "123",
		SessionId:     "ABCDE",
		RemoteAddress: "127.0.0.1",
	}
	expectedPayload := Payload{
		Type:  "string",
		Value: "Test 1",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarA", variantSetup.Name, "Should return VarA")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
	suite.Equal(expectedPayload, variantSetup.Payload, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_OverrideOnCustomProperties() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithOverride,
	}
	mockContext := &context.Context{
		Properties: map[string]string{
			"env": "dev",
		},
	}
	expectedPayload := Payload{
		Type:  "string",
		Value: "Test 3",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarC", variantSetup.Name, "Should return VarC")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
	suite.Equal(expectedPayload, variantSetup.Payload, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_ShouldReturnVarD() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithoutOverride,
	}
	mockContext := &context.Context{
		UserId: "123",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarE", variantSetup.Name, "Should return VarE")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_ShouldReturnVarE() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithoutOverride,
	}
	mockContext := &context.Context{
		UserId: "163",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarF", variantSetup.Name, "Should return VarF")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
}

func (suite *VariantTestSuite) TestGetVariant_ShouldReturnVarF() {
	mockFeature := Feature{
		Name:     "test.variants",
		Enabled:  true,
		Variants: suite.VariantWithoutOverride,
	}
	mockContext := &context.Context{
		UserId: "40",
	}
	variantSetup := VariantCollection{
		GroupId:  mockFeature.Name,
		Variants: mockFeature.Variants,
	}.GetVariant(mockContext)
	suite.Equal("VarE", variantSetup.Name, "Should return VarE")
	suite.Equal(true, variantSetup.Enabled, "Should be equal")
}

func TestVariantSuite(t *testing.T) {
	ts := VariantTestSuite{}
	suite.Run(t, &ts)
}
