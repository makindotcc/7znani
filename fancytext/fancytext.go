package fancytext

import "strings"

var alphabetNormal = strings.Split("a ą b c ć d e ę f g h i j k l ł m n ń o ó p r s ś t u w y z ź ż", " ")
var alphabetFancy = strings.Split("ᴀ ą ʙ ᴄ ć ᴅ ᴇ ę ғ ɢ ʜ ɪ ᴊ ᴋ ʟ ł ᴍ ɴ ń ᴏ ó ᴘ ʀ s ś ᴛ ᴜ ᴡ ʏ ᴢ ź ż", " ")
var alphabetMap map[int32]int32

func init() {
	alphabetMap = make(map[int32]int32, len(alphabetNormal))

	for i, letterNormal := range alphabetNormal {
		for _, e := range letterNormal {
			for _, v := range alphabetFancy[i] {
				alphabetMap[e] = v
			}
		}
	}
}

func Reverse(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

func Make(text string) string {
	textRune := []rune(text)
	for i, letter := range textRune {
		fancyLetter, ok := alphabetMap[letter]
		if !ok {
			continue
		}
		textRune[i] = fancyLetter
	}

	return string(textRune)
}
