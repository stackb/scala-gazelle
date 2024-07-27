package wildcardimport

import (
	"bufio"
	"bytes"
	"log"
	"regexp"
	"sort"
	"strings"
)

// omnistac/gum/testutils/DbDataInitUtils.scala:98: error: [rewritten by -quickfix] not found: value FixSessionDao
var notFoundLine = regexp.MustCompile(`^(.*):\d+: error: .*not found: (value|type) ([A-Z].*)$`)

type outputScanner struct {
	debug bool
}

func (s *outputScanner) scan(output []byte) ([]string, error) {
	notFound := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if s.debug {
			log.Println("line:", line)
		}
		if match := notFoundLine.FindStringSubmatch(line); match != nil {
			typeOrValue := match[3]
			notFound[typeOrValue] = true
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	list := make([]string, 0, len(notFound))
	for k := range notFound {
		list = append(list, k)
	}
	sort.Strings(list)

	return list, nil
}
