package reader

type Format string

const (
	FormatCSV  Format = "csv"
	FormatJSON Format = "json"
	FormatXML  Format = "xml"
)

type Input struct {
	Source string
	Kind   string
	Format Format
	Path   string
}
