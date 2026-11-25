package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const CorpusSHA256 = "9c871e809b082836339c665aa611476602301f3e6425eca0ded34185d8093f72"

var PosLabels = []string{"adjective", "adverb", "auxiliary verb", "conjunction", "definite article", "determiner", "exclam", "indefinite article", "infinitive marker", "linking verb", "modal verb", "noun", "number", "preposition", "pronoun", "verb"}

type Record struct {
	ID           int    `json:"id"`
	Word         string `json:"word"`
	POS          string `json:"pos"`
	Level        string `json:"level"`
	Section      string `json:"section"`
	JP           string `json:"jp"`
	Example      string `json:"example"`
	Letter       string `json:"letter"`
	search       string
	posCode      int
	levelCode    int
	sectionCodes []int
	cloze        *Span
}

type Span struct{ Start, Length int }
type Corpus struct {
	Rows          []*Record
	ByID          map[int]*Record
	SHA256        string
	ClozeEligible int
}

func normalize(s string) string { return strings.ToLower(norm.NFKC.String(s)) }

func LoadCorpus(path string, expected int) (*Corpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	r := csv.NewReader(io.TeeReader(f, h))
	r.FieldsPerRecord = 7
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	if strings.Join(header, ",") != "ID,WORD,POS,LEVEL,SECTION,JP,EXAMPLE" {
		return nil, fmt.Errorf("CSV header must be ID,WORD,POS,LEVEL,SECTION,JP,EXAMPLE")
	}
	c := &Corpus{Rows: []*Record{}, ByID: map[int]*Record{}}
	seen := map[string]bool{}
	for {
		v, e := r.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("CSV row %d: %w", len(c.Rows)+2, e)
		}
		id, e := strconv.Atoi(v[0])
		if e != nil || id < 1 {
			return nil, fmt.Errorf("invalid Corpus ID %q", v[0])
		}
		if id != len(c.Rows)+1 {
			return nil, fmt.Errorf("Corpus IDs must be ordered 1..N; got %d after %d", id, len(c.Rows))
		}
		if c.ByID[id] != nil {
			return nil, fmt.Errorf("duplicate Corpus ID %d", id)
		}
		if seen[v[1]+"\x00"+v[2]] {
			return nil, fmt.Errorf("duplicate WORD/POS at ID %d", id)
		}
		seen[v[1]+"\x00"+v[2]] = true
		if v[1] == "" || v[5] == "" || v[6] == "" {
			return nil, fmt.Errorf("empty required Corpus content at ID %d", id)
		}
		row := &Record{ID: id, Word: v[1], POS: v[2], Level: v[3], Section: v[4], JP: v[5], Example: v[6]}
		for i, p := range PosLabels {
			if row.POS == p {
				row.posCode = i + 1
			}
		}
		if row.posCode == 0 {
			return nil, fmt.Errorf("unsupported POS %q", row.POS)
		}
		switch row.Level {
		case "3000":
			row.levelCode = 1
		case "5000":
			row.levelCode = 2
		case "":
			row.levelCode = 3
		default:
			return nil, fmt.Errorf("invalid LEVEL %q", row.Level)
		}
		if row.Section == "" {
			row.sectionCodes = []int{19}
		} else {
			for _, section := range strings.Split(row.Section, " | ") {
				code := 0
				if len(section) >= 2 {
					n, _ := strconv.Atoi(section[1:])
					if section[0] == 'S' && n >= 1 && n <= 6 {
						code = n
					}
					if section[0] == 'W' && n >= 1 && n <= 12 {
						code = n + 6
					}
				}
				if code == 0 {
					return nil, fmt.Errorf("invalid SECTION %q at ID %d", section, id)
				}
				row.sectionCodes = append(row.sectionCodes, code)
			}
		}
		letter := []rune(norm.NFKC.String(row.Word))
		if len(letter) > 0 {
			row.Letter = string(unicode.ToUpper(letter[0]))
		}
		row.search = normalize(strings.Join(v[1:], " "))
		row.cloze = locateCloze(row)
		if row.cloze != nil {
			c.ClozeEligible++
		}
		c.Rows = append(c.Rows, row)
		c.ByID[id] = row
	}
	if expected > 0 && len(c.Rows) != expected {
		return nil, fmt.Errorf("expected %d Corpus records; got %d", expected, len(c.Rows))
	}
	c.SHA256 = hex.EncodeToString(h.Sum(nil))
	return c, nil
}

func (c *Corpus) Match(row *Record, f Filter) bool {
	if len(f.CorpusIDs) > 0 && !contains(f.CorpusIDs, row.ID) {
		return false
	}
	if f.InitialLetter > 0 && row.Letter != string(rune('A'+f.InitialLetter-1)) {
		return false
	}
	if len(f.POSCodes) > 0 && !contains(f.POSCodes, row.posCode) {
		return false
	}
	if len(f.LevelCodes) > 0 && !contains(f.LevelCodes, row.levelCode) {
		return false
	}
	if len(f.SectionCodes) > 0 {
		matched := false
		for _, s := range row.sectionCodes {
			if contains(f.SectionCodes, s) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, term := range strings.Fields(normalize(f.FreeText)) {
		if !strings.Contains(row.search, term) {
			return false
		}
	}
	return true
}

func contains(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

var headnote = regexp.MustCompile(`\s*\([^)]*\)|[⁰¹²³⁴⁵⁶⁷⁸⁹]+|[0-9]+$`)
var irregularForms = map[string][]string{
	"arise": {"arose", "arisen"}, "awake": {"awoke", "awoken"}, "be": {"am", "is", "are", "was", "were", "been", "being"},
	"bear": {"bore", "borne", "born"}, "beat": {"beaten"}, "become": {"became"}, "begin": {"began", "begun"}, "bend": {"bent"},
	"bet": {"bet"}, "bind": {"bound"}, "bite": {"bit", "bitten"}, "bleed": {"bled"}, "blow": {"blew", "blown"},
	"break": {"broke", "broken"}, "breed": {"bred"}, "bring": {"brought"}, "build": {"built"}, "burn": {"burnt"}, "burst": {"burst"},
	"buy": {"bought"}, "catch": {"caught"}, "choose": {"chose", "chosen"}, "cling": {"clung"}, "come": {"came"}, "cost": {"cost"},
	"creep": {"crept"}, "cut": {"cut"}, "deal": {"dealt"}, "dig": {"dug"}, "do": {"does", "did", "done"}, "draw": {"drew", "drawn"},
	"dream": {"dreamt"}, "drink": {"drank", "drunk"}, "drive": {"drove", "driven"}, "eat": {"ate", "eaten"}, "fall": {"fell", "fallen"},
	"feed": {"fed"}, "feel": {"felt"}, "fight": {"fought"}, "find": {"found"}, "flee": {"fled"}, "fly": {"flew", "flown"},
	"forbid": {"forbade", "forbidden"}, "forget": {"forgot", "forgotten"}, "forgive": {"forgave", "forgiven"}, "freeze": {"froze", "frozen"},
	"get": {"got", "gotten"}, "give": {"gave", "given"}, "go": {"goes", "went", "gone"}, "grow": {"grew", "grown"}, "hang": {"hung"},
	"have": {"has", "had"}, "hear": {"heard"}, "hide": {"hid", "hidden"}, "hit": {"hit"}, "hold": {"held"}, "hurt": {"hurt"},
	"keep": {"kept"}, "kneel": {"knelt"}, "know": {"knew", "known"}, "lay": {"laid"}, "lead": {"led"}, "lean": {"leant"},
	"learn": {"learnt"}, "leave": {"left"}, "lend": {"lent"}, "let": {"let"}, "lie": {"lay", "lain", "lying"}, "light": {"lit"},
	"lose": {"lost"}, "make": {"made"}, "mean": {"meant"}, "meet": {"met"}, "pay": {"paid"}, "read": {"read"}, "ride": {"rode", "ridden"},
	"ring": {"rang", "rung"}, "rise": {"rose", "risen"}, "run": {"ran"}, "say": {"said"}, "see": {"saw", "seen"}, "seek": {"sought"},
	"sell": {"sold"}, "send": {"sent"}, "set": {"set"}, "shake": {"shook", "shaken"}, "shine": {"shone"}, "shoot": {"shot"},
	"show": {"shown"}, "shrink": {"shrank", "shrunk"}, "shut": {"shut"}, "sing": {"sang", "sung"}, "sink": {"sank", "sunk"},
	"sit": {"sat"}, "sleep": {"slept"}, "slide": {"slid"}, "smell": {"smelt"}, "speak": {"spoke", "spoken"}, "spell": {"spelt"},
	"spend": {"spent"}, "spill": {"spilt"}, "spin": {"spun"}, "split": {"split"}, "spread": {"spread"}, "spring": {"sprang", "sprung"},
	"stand": {"stood"}, "steal": {"stole", "stolen"}, "stick": {"stuck"}, "sting": {"stung"}, "strike": {"struck", "stricken"},
	"swear": {"swore", "sworn"}, "sweep": {"swept"}, "swim": {"swam", "swum"}, "swing": {"swung"}, "take": {"took", "taken"},
	"teach": {"taught"}, "tear": {"tore", "torn"}, "tell": {"told"}, "think": {"thought"}, "throw": {"threw", "thrown"},
	"understand": {"understood"}, "wake": {"woke", "woken"}, "wear": {"wore", "worn"}, "weep": {"wept"}, "win": {"won"},
	"withdraw": {"withdrew", "withdrawn"}, "write": {"wrote", "written"}, "can": {"could"}, "may": {"might"}, "will": {"would"},
	"shall": {"should"}, "child": {"children"}, "foot": {"feet"}, "tooth": {"teeth"}, "mouse": {"mice"}, "man": {"men"},
	"woman": {"women"}, "person": {"people"}, "goose": {"geese"}, "good": {"better", "best"}, "bad": {"worse", "worst"},
	"half": {"halves"}, "leap": {"leapt"}, "overcome": {"overcame"}, "safe": {"safer", "safest"},
	"speed": {"sped"}, "undergo": {"underwent", "undergone"}, "undertake": {"undertook", "undertaken"},
	"uphold": {"upheld"}, "weave": {"wove", "woven"},
}

// A finite, conservative surface-form matcher. Never blanks an arbitrary word
// when the target cannot be located. Spans always index ORIGINAL example runes.
func locateCloze(row *Record) *Span {
	base := strings.TrimSpace(headnote.ReplaceAllString(row.Word, ""))
	parts := strings.FieldsFunc(base, func(r rune) bool { return r == ',' || r == '/' })
	forms := []string{}
	for _, part := range parts {
		b := strings.ToLower(strings.TrimSpace(part))
		if b == "" {
			continue
		}
		forms = append(forms, b)
		if row.POS == "verb" || row.POS == "auxiliary verb" || row.POS == "linking verb" || row.POS == "modal verb" || row.POS == "noun" || row.POS == "adjective" {
			forms = append(forms, irregularForms[b]...)
			if !strings.Contains(b, " ") {
				forms = append(forms, b+"s", b+"es")
				if strings.HasSuffix(b, "y") && len(b) > 1 {
					forms = append(forms, b[:len(b)-1]+"ies")
				}
				if strings.Contains(row.POS, "verb") {
					forms = append(forms, b+"ed", b+"ing")
					if strings.HasSuffix(b, "e") {
						forms = append(forms, b+"d", b[:len(b)-1]+"ing")
					}
					if strings.HasSuffix(b, "y") {
						forms = append(forms, b[:len(b)-1]+"ied")
					}
					if len(b) >= 3 && !strings.ContainsRune("aeiouwxy", rune(b[len(b)-1])) && strings.ContainsRune("aeiou", rune(b[len(b)-2])) {
						forms = append(forms, b+b[len(b)-1:]+"ed", b+b[len(b)-1:]+"ing")
					}
				}
			}
		}
	}
	ex := []rune(strings.ToLower(row.Example))
	for _, form := range forms {
		word := []rune(form)
		for i := 0; i+len(word) <= len(ex); i++ {
			if string(ex[i:i+len(word)]) != form {
				continue
			}
			if i > 0 && unicode.IsLetter(ex[i-1]) {
				continue
			}
			end := i + len(word)
			if end < len(ex) && unicode.IsLetter(ex[end]) {
				continue
			}
			return &Span{Start: i, Length: len(word)}
		}
	}
	return nil
}
