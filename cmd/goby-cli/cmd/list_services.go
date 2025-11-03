package cmd

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

// listServicesCmd represents the list-services command
var listServicesCmd = &cobra.Command{
	Use:   "list-services",
	Short: "Lists all services discoverable via the service registry",
	Long: `Scans the codebase for definitions of registry.Key[...] to find all services
that modules can resolve at runtime. This provides a live view of the
framework's shared services.`,
	Run: func(cmd *cobra.Command, args []string) {
		services, err := findRegistryKeys("./")
		if err != nil {
			log.Fatalf("Failed to find registry keys: %v", err)
		}

		if len(services) == 0 {
			fmt.Println("No services found in the registry.")
			return
		}

		printServices(services)
	},
}

func init() {
	rootCmd.AddCommand(listServicesCmd)
}

type ServiceInfo struct {
	Key  string
	Type string
}

// findRegistryKeys scans the project directory for registry.Key definitions.
func findRegistryKeys(root string) ([]ServiceInfo, error) {
	var services []ServiceInfo

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   root,
		Tests: false, // Don't include test files
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				// We are looking for `var MyKey = registry.Key...`
				genDecl, ok := n.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.VAR {
					return true
				}

				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok || len(valueSpec.Values) != 1 {
						continue
					}

					// Check if the value is a call expression, e.g., Key(...)
					callExpr, ok := valueSpec.Values[0].(*ast.CallExpr)
					if !ok {
						continue
					}

					// Check if the function being called is a generic type instantiation, e.g., Key[Type]
					indexListExpr, ok := callExpr.Fun.(*ast.IndexListExpr)
					if !ok {
						continue
					}

					// Use the type checker to see if this is the correct `registry.Key`
					typeObj := pkg.TypesInfo.TypeOf(indexListExpr.X)
					if typeObj == nil {
						continue
					}

					// Check if the type's name is `Key` and its package path is correct.
					named, ok := typeObj.(*types.Named)
					if !ok || named.Obj().Name() != "Key" || !strings.HasSuffix(named.Obj().Pkg().Path(), "internal/registry") {
						continue
					}

					// We found a registry.Key definition!
					service := ServiceInfo{}

					// Extract the string key from the first argument
					if len(callExpr.Args) == 1 {
						if keyLit, ok := callExpr.Args[0].(*ast.BasicLit); ok && keyLit.Kind == token.STRING {
							service.Key = keyLit.Value[1 : len(keyLit.Value)-1] // Trim quotes
						}
					}

					// Extract the type parameter
					if len(indexListExpr.Indices) == 1 {
						// Use types.ExprString for a clean representation of the type
						service.Type = types.ExprString(indexListExpr.Indices[0])
					}

					services = append(services, service)
				}
				return true
			})
		}
	}

	return services, nil
}

func printServices(services []ServiceInfo) {
	fmt.Println("Available Services in the Registry:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "KEY\tTYPE")
	fmt.Fprintln(w, "---\t----")
	for _, s := range services {
		fmt.Fprintf(w, "%s\t%s\n", s.Key, s.Type)
	}
	w.Flush()
}
