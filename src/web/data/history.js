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
});
