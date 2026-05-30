// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

function isLocalStoageSupported() {
  if (typeof localStorage === 'object') {
    try {
      localStorage.setItem("__local_storage_test__", 1);
      localStorage.removeItem("__local_storage_test__");
      return true;
    } catch (e) {
      return false;
    }
  }

  return false;
}

function historyRefreshList() {
  const shortMonth = [{{call .Translate `pasteJS.ShortMonth`}}];

  // Get and clean list
  let listElement = document.getElementById("js-history-popup-list");
  listElement.innerHTML = "";

  // Read locale storage
  let timeNowUnix = Math.floor(Date.now() / 1000)
  let historyJSON = localStorage.getItem("history");
  if (historyJSON != null) {
    let history = JSON.parse(historyJSON);

    for (let i = 0; history.length > i; i++) {
      // Check title
      let title = history[i].title;
      if (title == "") {
        title = "{{ call .Translate `historyJS.Untitled` }}";
      }

      // Convert create date to string
      let date = new Date(history[i].createTime * 1000)

      let dateDayStr = date.getDate();
      if (date.getDate() < 10) {
        dateDayStr = "0" + dateDayStr;
      }
      let dateStr = dateDayStr + " " + shortMonth[date.getMonth()] + ", " + date.getFullYear();

      // Add row
      if (timeNowUnix < history[i].deleteTime || history[i].deleteTime == 0) {
        listElement.insertAdjacentHTML("beforeend", "<li>[" + dateStr + "] <a href='/"+history[i].id+"'>"+title+"</a></li>");
      } else {
        listElement.insertAdjacentHTML("beforeend", "<li><del>[" + dateStr + "] <a class='text-grey' href='/"+history[i].id+"'>"+title+"</a></del></li>");
      }
    }
  }
}

function historyPopUpShow() {
  document.getElementById('history-popup-state').checked = true;
  historyRefreshList();
}

function historyPopUpHide() {
  document.getElementById('history-popup-state').checked = false;
}

function historyPopUpEscEvent(event) {
  // If ESC pressed
  if (event.keyCode == 27) {
    historyPopUpHide();
  }
}

function historyEnable() {
  if (document.getElementById("js-history-popup-enable").checked == true) {
    localStorage.removeItem("DisableHistory");
    showToast("{{ call .Translate `historyJS.HistoryEnabledAlert` }}", "success");
  } else {
    localStorage.setItem("DisableHistory", true);
    showToast("{{ call .Translate `historyJS.HistoryDisabledAlert` }}", "info");
  }
}

function historyClear() {
  showConfirm(
    "{{ call .Translate `historyJS.ClearHistoryConfirm` }}",
    function() {
      localStorage.removeItem("history");
      historyRefreshList();
    },
    null,
    "{{ call .Translate `pasteContinue.Cancel` }}",
    "{{ call .Translate `pasteContinue.Continue` }}"
  );
}

document.addEventListener("DOMContentLoaded", () => {
  // Attach event listeners to server-rendered popup elements
  document.getElementById("js-history-popup-enable").addEventListener("change", historyEnable);
  document.getElementById("js-history-popup-clear").addEventListener("click", historyClear);

  // Set "Remember history" checkbox state
  document.getElementById("js-history-popup-enable").checked = !localStorage.getItem("DisableHistory");

  // Refresh list when popup is opened via checkbox
  document.getElementById('history-popup-state').addEventListener('change', function() {
    if (this.checked) {
      historyRefreshList();
    }
  });

  // Add ESC key handler when popup is open
  document.addEventListener("keydown", function(event) {
    if (event.keyCode == 27 && document.getElementById('history-popup-state').checked) {
      historyPopUpHide();
    }
  });

  // If exist "create paste" form path it
  let createPasteForm = document.getElementById("create-paste-form");
  if (createPasteForm != null) {
    createPasteForm.addEventListener("submit", (event) => {
      event.preventDefault();

      let fileInput = document.getElementById("paste-file");
      let hasFile = fileInput && fileInput.files && fileInput.files.length > 0;

      // Read the CSRF token from the hidden form field
      let csrfTokenEl = createPasteForm.querySelector('input[name="csrf_token"]');
      let csrfToken = csrfTokenEl ? csrfTokenEl.value : "";

      // Get the paste title for history storage
      let title = "";
      let titleEl = createPasteForm.querySelector('input[name="title"],textarea[name="title"]');
      if (titleEl) {
        title = titleEl.value;
      }

      if (hasFile) {
        // File upload: use FormData so the binary payload is preserved.
        // Send to / (web handler) with the CSRF token in the header so the
        // middleware finds it without having to parse the multipart body first.
        var xhrFile = new XMLHttpRequest();
        xhrFile.open("POST", "/", true);
        xhrFile.setRequestHeader("X-CSRF-Token", csrfToken);

        // Fall back to native form submit on network error so paste creation still works
        xhrFile.onerror = () => { createPasteForm.submit(); };

        xhrFile.onload = () => {
          if (xhrFile.status != 200) {
            switch (xhrFile.status) {
              case 400: showToast("{{ call .Translate `error.400` | call .Translate `historyJS.Error` 400 }}", "error"); break;
              case 401: showToast("{{ call .Translate `error.401` | call .Translate `historyJS.Error` 401 }}", "error"); break;
              case 403: showToast("{{ call .Translate `error.403` | call .Translate `historyJS.Error` 403 }}", "error"); break;
              case 413: showToast("{{ call .Translate `error.413` | call .Translate `historyJS.Error` 413 }}", "error"); break;
              case 429: showToast("{{ call .Translate `error.429` | call .Translate `historyJS.Error` 429 }}", "warning"); break;
              case 500: showToast("{{ call .Translate `error.500` | call .Translate `historyJS.Error` 500 }}", "error"); break;
              default: showToast("{{ call .Translate `historyJS.ErrorUnknown` `"+xhrFile.status+"` }}", "error"); break;
            }
            return;
          }

          // After following the server redirect, responseURL is the paste URL
          var pasteURL = xhrFile.responseURL;
          var pasteId = pasteURL.replace(/\/+$/, "").split("/").pop();

          // Save to history
          if (localStorage.getItem("DisableHistory") != "true") {
            let historyJSON = localStorage.getItem("history");
            let history = [];
            if (historyJSON != null) {
              history = JSON.parse(historyJSON);
            }
            history.splice(0, 0, {id: pasteId, createTime: Math.floor(Date.now() / 1000), deleteTime: 0, title: title});
            localStorage.setItem("history", JSON.stringify(history));
          }

          window.location = pasteURL;
        };

        xhrFile.send(new FormData(createPasteForm));
        return false;
      }

      // Text paste: encode as URL-encoded and POST to the CSRF-exempt API
      let data = "";
      Array.from(createPasteForm.elements)
        .filter((item) => !!item.name && item.type !== "file")
        .map((element) => {
          let { name, value, type } = element;

          if (type == "checkbox") {
            if (element.checked) {
              value = "true";
            } else {
              value = "false";
            }
          }

          data = data + "&" + name + "=" + encodeURIComponent(value);
        })
      data = data.slice(1);

      // Send request
      var xhr = new XMLHttpRequest();
      xhr.responseType = "json";
      xhr.open("POST", "/api/v1/pastes", true);
      xhr.setRequestHeader("Content-type", "application/x-www-form-urlencoded");

      // Fall back to native form submit on network error so paste creation still works
      xhr.onerror = () => { createPasteForm.submit(); };

      xhr.onload = () => {
        // Check HTTP code
        if (xhr.status != 200) {
          switch (xhr.status) {
            case 400: showToast("{{ call .Translate `error.400` | call .Translate `historyJS.Error` 400 }}", "error"); break;
            case 401: showToast("{{ call .Translate `error.401` | call .Translate `historyJS.Error` 401 }}", "error"); break;
            case 404: showToast("{{ call .Translate `error.404` | call .Translate `historyJS.Error` 404 }}", "error"); break;
            case 405: showToast("{{ call .Translate `error.405` | call .Translate `historyJS.Error` 405 }}", "error"); break;
            case 413: showToast("{{ call .Translate `error.413` | call .Translate `historyJS.Error` 413 }}", "error"); break;
            case 429: showToast("{{ call .Translate `error.429` | call .Translate `historyJS.Error` 429 }}", "warning"); break;
            case 500: showToast("{{ call .Translate `error.500` | call .Translate `historyJS.Error` 500 }}", "error"); break;
            default: showToast("{{ call .Translate `historyJS.ErrorUnknown` `"+xhr.status+"` }}", "error"); break;
          }
          return;
        }

        // API response is {"ok": true, "data": {"id": "...", ...}}
        var paste = (xhr.response && xhr.response.data) ? xhr.response.data : xhr.response;

        // Guard against malformed or missing response — fall back to native submit
        if (!paste || !paste.id) {
          createPasteForm.submit();
          return;
        }

        // Save to history
        if (localStorage.getItem("DisableHistory") != "true") {
          let historyJSON = localStorage.getItem("history");
          let history = [];
          if (historyJSON != null) {
            history = JSON.parse(historyJSON);
          }

          history.splice(0, 0, {id: paste.id, createTime: paste.createTime, deleteTime: paste.deleteTime, title: title});
          localStorage.setItem("history", JSON.stringify(history));
        }

        // Redirect to the new paste
        window.location = "/" + paste.id;
      };

      xhr.send(data);

      return false;
    });
  }
});
