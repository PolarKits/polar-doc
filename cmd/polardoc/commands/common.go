package commands

import "github.com/PolarKits/polar-doc/internal/doc"

func parseDocumentRef(command string, args []string) (doc.DocumentRef, error) {
	input, err := parseCommandInput(command, args)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	return doc.DocumentRef{
		Format: format,
		Path:   input.path,
	}, nil
}

func detectFormatByExtension(path string) (doc.Format, error) {
	return doc.DetectFormatByExtension(path)
}
