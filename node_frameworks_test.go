package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NodeFrameworksTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *NodeFrameworksTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
	os.Chdir(suite.helper.TempDir())
}

func (suite *NodeFrameworksTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func TestNodeFrameworksSuite(t *testing.T) {
	suite.Run(t, new(NodeFrameworksTestSuite))
}

// TestDetectNodeFramework tests framework detection from package.json
func (suite *NodeFrameworksTestSuite) TestDetectNodeFramework() {
	testCases := []struct {
		name              string
		packageJSON       string
		expectedFramework string
	}{
		{
			name: "Next.js",
			packageJSON: `{
				"dependencies": {
					"next": "14.0.0",
					"react": "18.0.0"
				}
			}`,
			expectedFramework: "nextjs",
		},
		{
			name: "Nuxt",
			packageJSON: `{
				"dependencies": {
					"nuxt": "3.0.0",
					"vue": "3.0.0"
				}
			}`,
			expectedFramework: "nuxt",
		},
		{
			name: "Angular",
			packageJSON: `{
				"dependencies": {
					"@angular/core": "17.0.0",
					"@angular/common": "17.0.0"
				}
			}`,
			expectedFramework: "angular",
		},
		{
			name: "Express",
			packageJSON: `{
				"dependencies": {
					"express": "4.18.0",
					"body-parser": "1.20.0"
				}
			}`,
			expectedFramework: "express",
		},
		{
			name: "NestJS",
			packageJSON: `{
				"dependencies": {
					"@nestjs/core": "10.0.0",
					"@nestjs/common": "10.0.0"
				}
			}`,
			expectedFramework: "nestjs",
		},
		{
			name: "React (without Next.js)",
			packageJSON: `{
				"dependencies": {
					"react": "18.0.0",
					"react-dom": "18.0.0"
				}
			}`,
			expectedFramework: "react",
		},
		{
			name: "Vue (without Nuxt)",
			packageJSON: `{
				"dependencies": {
					"vue": "3.0.0",
					"vue-router": "4.0.0"
				}
			}`,
			expectedFramework: "vue",
		},
		{
			name: "Svelte",
			packageJSON: `{
				"devDependencies": {
					"svelte": "4.0.0",
					"@sveltejs/kit": "2.0.0"
				}
			}`,
			expectedFramework: "svelte",
		},
		{
			name: "Fastify",
			packageJSON: `{
				"dependencies": {
					"fastify": "4.0.0"
				}
			}`,
			expectedFramework: "fastify",
		},
		{
			name: "Remix",
			packageJSON: `{
				"dependencies": {
					"@remix-run/node": "2.0.0",
					"@remix-run/react": "2.0.0"
				}
			}`,
			expectedFramework: "remix",
		},
		{
			name: "No framework",
			packageJSON: `{
				"dependencies": {
					"lodash": "4.17.21"
				}
			}`,
			expectedFramework: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create test directory with package.json
			testDir := "test-" + tc.name
			packagePath := filepath.Join(testDir, "package.json")
			suite.helper.CreateFile(packagePath, tc.packageJSON)

			// Detect framework
			framework := detectNodeFramework(testDir)
			suite.Equal(tc.expectedFramework, framework)
		})
	}
}

// TestHasPackage tests the package detection helper
func (suite *NodeFrameworksTestSuite) TestHasPackage() {
	pkg := PackageJSON{
		Dependencies: map[string]string{
			"express":    "4.18.0",
			"body-parser": "1.20.0",
		},
		DevDependencies: map[string]string{
			"nodemon": "3.0.0",
			"jest":    "29.0.0",
		},
	}

	// Test dependencies
	suite.True(hasPackage(pkg, "express"))
	suite.True(hasPackage(pkg, "body-parser"))
	
	// Test devDependencies
	suite.True(hasPackage(pkg, "nodemon"))
	suite.True(hasPackage(pkg, "jest"))
	
	// Test non-existent package
	suite.False(hasPackage(pkg, "react"))
	suite.False(hasPackage(pkg, "vue"))
}

// TestGetStartScriptFromPackageJSON tests extracting start script
func (suite *NodeFrameworksTestSuite) TestGetStartScriptFromPackageJSON() {
	testCases := []struct {
		name           string
		packageJSON    string
		expectedScript string
	}{
		{
			name: "With start script",
			packageJSON: `{
				"scripts": {
					"start": "node server.js",
					"dev": "nodemon server.js"
				}
			}`,
			expectedScript: "node server.js",
		},
		{
			name: "With dev script only",
			packageJSON: `{
				"scripts": {
					"dev": "next dev",
					"build": "next build"
				}
			}`,
			expectedScript: "next dev",
		},
		{
			name: "With serve script",
			packageJSON: `{
				"scripts": {
					"serve": "vue-cli-service serve",
					"build": "vue-cli-service build"
				}
			}`,
			expectedScript: "vue-cli-service serve",
		},
		{
			name: "With main field",
			packageJSON: `{
				"main": "index.js",
				"scripts": {
					"test": "jest"
				}
			}`,
			expectedScript: "node index.js",
		},
		{
			name: "No start script",
			packageJSON: `{
				"scripts": {
					"test": "jest",
					"lint": "eslint ."
				}
			}`,
			expectedScript: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			testDir := "test-" + tc.name
			packagePath := filepath.Join(testDir, "package.json")
			suite.helper.CreateFile(packagePath, tc.packageJSON)

			script := getStartScriptFromPackageJSON(testDir)
			suite.Equal(tc.expectedScript, script)
		})
	}
}

// TestGetBuildScriptFromPackageJSON tests extracting build script
func (suite *NodeFrameworksTestSuite) TestGetBuildScriptFromPackageJSON() {
	testCases := []struct {
		name           string
		packageJSON    string
		expectedScript string
	}{
		{
			name: "With build script",
			packageJSON: `{
				"scripts": {
					"build": "webpack --mode production",
					"start": "node server.js"
				}
			}`,
			expectedScript: "webpack --mode production",
		},
		{
			name: "With compile script",
			packageJSON: `{
				"scripts": {
					"compile": "tsc",
					"start": "node dist/index.js"
				}
			}`,
			expectedScript: "tsc",
		},
		{
			name: "No build script",
			packageJSON: `{
				"scripts": {
					"start": "node server.js",
					"test": "jest"
				}
			}`,
			expectedScript: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			testDir := "test-" + tc.name
			packagePath := filepath.Join(testDir, "package.json")
			suite.helper.CreateFile(packagePath, tc.packageJSON)

			script := getBuildScriptFromPackageJSON(testDir)
			suite.Equal(tc.expectedScript, script)
		})
	}
}

// TestGetFrameworkCommand tests framework-specific commands
func (suite *NodeFrameworksTestSuite) TestGetFrameworkCommand() {
	testCases := []struct {
		framework string
		pm        string
		isDev     bool
		expected  string
	}{
		// Next.js
		{"nextjs", "npm", true, "npm run dev"},
		{"nextjs", "npm", false, "npm run start"},
		{"nextjs", "yarn", true, "yarn dev"},
		{"nextjs", "pnpm", false, "pnpm start"},
		
		// Angular
		{"angular", "npm", true, "npm run start"},
		{"angular", "npm", false, "npm run serve"},
		
		// React
		{"react", "npm", true, "npm run start"},
		{"react", "npm", false, "npm run serve"},
		{"react", "yarn", true, "yarn start"},
		
		// Vue
		{"vue", "npm", true, "npm run serve"},
		{"vue", "npm", false, "npm run preview"},
		
		// Express
		{"express", "npm", true, "npm run dev"},
		{"express", "npm", false, "npm run start"},
		
		// Unknown framework
		{"unknown", "npm", true, "npm run dev"},
		{"unknown", "npm", false, "npm run start"},
	}

	for _, tc := range testCases {
		name := tc.framework + "-" + tc.pm
		if tc.isDev {
			name += "-dev"
		} else {
			name += "-prod"
		}
		
		suite.Run(name, func() {
			cmd := getFrameworkCommand(tc.framework, tc.pm, tc.isDev)
			suite.Equal(tc.expected, cmd)
		})
	}
}

// TestGetPackageJSON tests reading and parsing package.json
func (suite *NodeFrameworksTestSuite) TestGetPackageJSON() {
	suite.Run("Valid package.json", func() {
		packageContent := `{
			"name": "test-app",
			"version": "1.0.0",
			"scripts": {
				"start": "node index.js"
			},
			"dependencies": {
				"express": "4.18.0"
			}
		}`
		
		suite.helper.CreateFile("valid/package.json", packageContent)
		
		pkg, err := getPackageJSON("valid")
		suite.NoError(err)
		suite.NotNil(pkg)
		suite.Equal("test-app", pkg.Name)
		suite.Equal("1.0.0", pkg.Version)
		suite.Equal("node index.js", pkg.Scripts["start"])
		suite.Equal("4.18.0", pkg.Dependencies["express"])
	})

	suite.Run("Missing package.json", func() {
		pkg, err := getPackageJSON("nonexistent")
		suite.NoError(err)
		suite.Nil(pkg)
	})

	suite.Run("Empty folder", func() {
		pkg, err := getPackageJSON("")
		suite.NoError(err)
		suite.Nil(pkg)
	})

	suite.Run("Invalid JSON", func() {
		suite.helper.CreateFile("invalid/package.json", "{ invalid json }")
		
		pkg, err := getPackageJSON("invalid")
		suite.Error(err)
		suite.Nil(pkg)
	})
}