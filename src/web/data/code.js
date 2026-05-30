// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

function copyToClipboard(text) {
  var tmp = document.createElement("textarea");
  var focus = document.activeElement;

  tmp.value = text;

  document.body.appendChild(tmp);
  tmp.select();
  document.execCommand("copy");
  document.body.removeChild(tmp);
  focus.focus();
}

function copyButton(element) {
  var result = "";

  var strings = element.parentNode.getElementsByTagName("code")[0].textContent.split("\n");
  var stringsLen = strings.length;
  var cutLen = stringsLen.toString().length;

  for (var i = 0; stringsLen > i; i++) {
    if (i !== 0) {
      result = result + "\n";
    }

    result = result + strings[i].slice(cutLen);
  }

  result = result.trim() + "\n";
  copyToClipboard(result);
}

document.addEventListener("DOMContentLoaded", function() {
  // Add copy button to all pre tags
  var preElements = document.getElementsByTagName("pre");

  for (var i = 0; preElements.length > i; i++) {
    var btn = document.createElement("button");
    btn.className = "button-green copy-btn";
    btn.textContent = "{{call .Translate `codeJS.Paste`}}";
    btn.addEventListener("click", function() {
      copyButton(this);
    });
    preElements[i].appendChild(btn);
  }
});
