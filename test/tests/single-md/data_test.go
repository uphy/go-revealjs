package singlemd

import (
	"testing"

	"github.com/uphy/go-revealjs/test/runner"
)

func Test(t *testing.T) {
	runner.Run(t, func(asserter *runner.BuildResultAsserter) {
		asserter.HasRevealJSFiles(t)
		asserter.NotHasFile(t, "*.md")

		indexHTML := asserter.IndexHTML(t)
		indexHTML.HasString(t, `<section data-markdown data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$"># Page 1
        
			foo
			
			---
			
			# Page 2
			
			bar</section>`)
		indexHTML.HasTitle(t, "reveal.js")
		indexHTML.HasTheme(t, "black")
		indexHTML.HasStandardScriptTags(t)
		indexHTML.HasConfigProperty(t, "plugins", `[
                                        RevealMarkdown,
                                        RevealHighlight,
                                        RevealSearch,
                                        RevealNotes,
                                        RevealMath,
                                        RevealZoom,
                                ]`)
	})
}
