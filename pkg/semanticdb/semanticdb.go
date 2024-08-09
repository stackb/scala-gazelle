package semanticdb

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/proto"

	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

func ReadJarFile(filename string) ([]*spb.TextDocuments, error) {
	jar, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("opening jar file: %v", err)
	}
	defer jar.Close()

	return ReadJar(jar)
}

func ReadJar(jar *zip.ReadCloser) ([]*spb.TextDocuments, error) {
	docs := make([]*spb.TextDocuments, 0)

	for _, file := range jar.File {

		if !strings.HasPrefix(file.Name, "META-INF/semanticdb/") {
			continue
		}
		if !strings.HasSuffix(file.Name, ".semanticdb") {
			continue
		}

		fmt.Println("File Name:", file.Name)

		if doc, err := ReadJarZipFile(file); err != nil {
			return nil, fmt.Errorf("reading file within jar: %v", err)
		} else {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

func ReadJarZipFile(file *zip.File) (*spb.TextDocuments, error) {
	fileReader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening file within jar: %v", err)
	}
	defer fileReader.Close()

	return ReadTextDocument(fileReader)
}

func ReadTextDocument(in io.ReadCloser) (*spb.TextDocuments, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	var doc spb.TextDocuments
	if err := proto.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
