function setupDailyTrigger() {
  // 既存トリガー削除（重複防止）
  ScriptApp.getProjectTriggers().forEach(function(trigger) {
    if (trigger.getHandlerFunction() === 'dailySync') {
      ScriptApp.deleteTrigger(trigger);
    }
  });

  ScriptApp.newTrigger('dailySync')
    .timeBased()
    .atHour(9)
    .everyDays(1)
    .inTimezone('Asia/Tokyo')
    .create();
}

function dailySync() {
  var yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  var dateString = Utilities.formatDate(yesterday, 'UTC', 'yyyy-MM-dd');

  try {
    var data = fetchDailyAnalytics(dateString);
    writeAnalyticsToSheet(data, dateString);
    logSync(dateString, 'success', data.length, '');
  } catch (e) {
    logSync(dateString, 'error', 0, e.message);
    MailApp.sendEmail(
      Session.getActiveUser().getEmail(),
      '[CC Dashboard] Daily sync failed',
      'Date: ' + dateString + '\nError: ' + e.message
    );
  }
}

function backfillData(startDateStr, endDateStr) {
  var current = new Date(startDateStr);
  var end = new Date(endDateStr);

  while (current <= end) {
    var dateStr = Utilities.formatDate(current, 'UTC', 'yyyy-MM-dd');
    if (!isAlreadySynced(dateStr)) {
      try {
        var data = fetchDailyAnalytics(dateStr);
        writeAnalyticsToSheet(data, dateStr);
        logSync(dateStr, 'success', data.length, '');
      } catch (e) {
        logSync(dateStr, 'error', 0, e.message);
      }
      Utilities.sleep(1000);
    }
    current.setDate(current.getDate() + 1);
  }
}

function isAlreadySynced(dateString) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('SyncLog');
  var data = sheet.getDataRange().getValues();
  return data.some(function(row) {
    return Utilities.formatDate(new Date(row[0]), 'UTC', 'yyyy-MM-dd') === dateString
      && row[1] === 'success';
  });
}

function logSync(dateString, status, recordCount, errorMessage) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('SyncLog');
  sheet.appendRow([dateString, status, recordCount, new Date(), errorMessage]);
}
