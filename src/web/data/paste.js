/**
 * This file is part of CasPaste.
 * CasPaste is free software released under the MIT License.
 * See LICENSE.md file for details.
 */

document.addEventListener("DOMContentLoaded", function() {
  var shortWeekDay = [{{call .Translate `pasteJS.ShortWeekDay`}}];
  var shortMonth = [{{call .Translate `pasteJS.ShortMonth`}}];

  function dateToString(date) {
    var dateStr = shortWeekDay[date.getDay()] + ", " + date.getDate() + " " + shortMonth[date.getMonth()];
    dateStr = dateStr + " " + date.getFullYear();
    dateStr = dateStr + " " + date.getHours() + ":" + date.getMinutes() + ":" + date.getSeconds();

    var tz = date.getTimezoneOffset() / 60 * -1;
    if (tz >= 0) {
      dateStr = dateStr + " +" + tz;
    } else {
      dateStr = dateStr + " " + tz;
    }

    return dateStr;
  }

  var createTime = document.getElementById("createTime");
  createTime.textContent = dateToString(new Date(createTime.textContent));

  var deleteTime = document.getElementById("deleteTime");
  if (deleteTime !== null) {
    deleteTime.textContent = dateToString(new Date(deleteTime.textContent));
  }
});
