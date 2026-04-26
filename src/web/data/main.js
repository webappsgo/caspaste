/**
 * This file is part of CasPaste.
 * CasPaste is free software released under the MIT License.
 * See LICENSE.md file for details.
 */

document.addEventListener("DOMContentLoaded", function() {
	var editor = document.getElementById("editor");

	// Handle TAB key in editor
	editor.addEventListener("keydown", function(e) {
		if (e.keyCode === 9) {
			e.preventDefault();

			var startOrig = editor.selectionStart;
			var endOrig = editor.selectionEnd;

			editor.value = editor.value.substring(0, startOrig) + "\t" + editor.value.substring(endOrig);
			editor.selectionStart = editor.selectionEnd = startOrig + 1;
		}
	});

	// Add HTML and CSS code for line numbers support
	var editorContainer = document.getElementById("editor-container");
	if (editorContainer) {
		editorContainer.insertAdjacentHTML("afterbegin", "<textarea id='editorLines' wrap='off' tabindex=-1 readonly>1</textarea>");
	} else {
		editor.insertAdjacentHTML("beforebegin", "<textarea id='editorLines' wrap='off' tabindex=-1 readonly>1</textarea>");
	}

	var editorLines = document.getElementById("editorLines");
	editorLines.rows = editor.rows;

	// Focus editor when line numbers clicked
	editorLines.addEventListener("focus", function() {
		editor.focus();
	});

	// Sync height of line numbers with editor
	function syncEditorHeight() {
		editorLines.style.height = editor.offsetHeight + "px";
	}
	syncEditorHeight();

	// Use ResizeObserver if available for dynamic height sync
	if (window.ResizeObserver) {
		new ResizeObserver(syncEditorHeight).observe(editor);
	}

	// Sync scroll position
	editor.addEventListener("scroll", function() {
		editorLines.scrollTop = editor.scrollTop;
		editorLines.scrollLeft = editor.scrollLeft;
	});

	// Update line numbers on input
	var lineCountCache = 0;
	editor.addEventListener("input", function() {
		var lineCount = editor.value.split("\n").length;

		if (lineCountCache !== lineCount) {
			editorLines.value = "";

			for (var i = 0; i < lineCount; i++) {
				editorLines.value = editorLines.value + (i + 1) + "\n";
			}

			lineCountCache = lineCount;
		}
	});

	// Add symbol counter
	var symbolCounterContainer = document.getElementById("symbolCounterContainer");
	if (symbolCounterContainer) {
		symbolCounterContainer.innerHTML = "<span id='symbolCounter' class='text-grey'></span>";
		var symbolCounter = document.getElementById("symbolCounter");

		function updateSymbolCounter() {
			var length = editor.value.length;

			if (editor.maxLength !== -1) {
				symbolCounter.textContent = length + "/" + editor.maxLength;
			} else {
				symbolCounter.textContent = length + "/∞";
			}
		}

		editor.addEventListener("input", updateSymbolCounter);
		updateSymbolCounter();
	}

	// Handle file upload and textarea mutual exclusivity
	var fileInput = document.getElementById("paste-file");
	var textarea = document.getElementById("editor");

	if (fileInput && textarea) {
		// When file is selected, disable textarea
		fileInput.addEventListener("change", function() {
			if (this.files && this.files.length > 0) {
				textarea.disabled = true;
				textarea.required = false;
				textarea.classList.add("disabled");
			} else {
				textarea.disabled = false;
				textarea.required = false;
				textarea.classList.remove("disabled");
			}
		});

		// When text is entered, disable file input
		textarea.addEventListener("input", function() {
			if (this.value.trim().length > 0) {
				fileInput.disabled = true;
			} else {
				fileInput.disabled = false;
			}
		});
	}
});
