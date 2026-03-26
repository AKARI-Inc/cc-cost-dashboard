function getAvailableDateRange() {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('RawData');
  var dates = sheet.getRange('A2:A' + sheet.getLastRow()).getValues().flat().filter(Boolean);
  if (dates.length === 0) return { min: null, max: null };
  dates.sort(function(a, b) { return new Date(a) - new Date(b); });
  return {
    min: Utilities.formatDate(new Date(dates[0]), 'UTC', 'yyyy-MM-dd'),
    max: Utilities.formatDate(new Date(dates[dates.length - 1]), 'UTC', 'yyyy-MM-dd')
  };
}

function getSummaryByUser(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var summary = {};

  rows.forEach(function(r) {
    var email = r[1];
    if (!summary[email]) {
      summary[email] = {
        email: email,
        totalCostCents: 0, totalTokensInput: 0, totalTokensOutput: 0,
        totalSessions: 0, linesAdded: 0, linesRemoved: 0,
        commits: 0, pullRequests: 0,
        editAccepted: 0, editRejected: 0,
        writeAccepted: 0, writeRejected: 0,
        days: new Set()
      };
    }
    var s = summary[email];
    s.totalCostCents += r[24] || 0;      // cost_cents
    s.totalTokensInput += r[20] || 0;    // tokens_input
    s.totalTokensOutput += r[21] || 0;   // tokens_output
    var dateKey = Utilities.formatDate(new Date(r[0]), 'UTC', 'yyyy-MM-dd');
    if (!s.days.has(dateKey)) {
      s.days.add(dateKey);
      s.totalSessions += r[6] || 0;      // num_sessions
      s.linesAdded += r[7] || 0;
      s.linesRemoved += r[8] || 0;
      s.commits += r[9] || 0;
      s.pullRequests += r[10] || 0;
      s.editAccepted += r[11] || 0;
      s.editRejected += r[12] || 0;
      s.writeAccepted += r[15] || 0;
      s.writeRejected += r[16] || 0;
    }
  });

  // Set は JSON.stringify できないので削除
  return Object.values(summary).map(function(s) {
    s.activeDays = s.days.size;
    delete s.days;
    return s;
  });
}

function getDailyCostTrend(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var trend = {};

  rows.forEach(function(r) {
    var dateKey = Utilities.formatDate(new Date(r[0]), 'UTC', 'yyyy-MM-dd');
    var email = r[1];
    var key = dateKey + '|' + email;
    if (!trend[key]) trend[key] = { date: dateKey, email: email, costCents: 0 };
    trend[key].costCents += r[24] || 0;
  });

  return Object.values(trend).sort(function(a, b) {
    return a.date.localeCompare(b.date) || a.email.localeCompare(b.email);
  });
}

function getModelBreakdown(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var models = {};

  rows.forEach(function(r) {
    var model = r[19] || 'unknown';
    if (!models[model]) models[model] = { model: model, costCents: 0, tokensInput: 0, tokensOutput: 0 };
    models[model].costCents += r[24] || 0;
    models[model].tokensInput += r[20] || 0;
    models[model].tokensOutput += r[21] || 0;
  });

  return Object.values(models).sort(function(a, b) { return b.costCents - a.costCents; });
}

function getDailyProductivity(startDate, endDate) {
  var rows = getFilteredRows(startDate, endDate);
  var daily = {};

  rows.forEach(function(r) {
    var dateKey = Utilities.formatDate(new Date(r[0]), 'UTC', 'yyyy-MM-dd');
    var email = r[1];
    var uniqueKey = dateKey + '|' + email;

    if (!daily[dateKey]) daily[dateKey] = { date: dateKey, sessions: 0, commits: 0, pullRequests: 0, seen: {} };
    if (!daily[dateKey].seen[uniqueKey]) {
      daily[dateKey].seen[uniqueKey] = true;
      daily[dateKey].sessions += r[6] || 0;
      daily[dateKey].commits += r[9] || 0;
      daily[dateKey].pullRequests += r[10] || 0;
    }
  });

  return Object.values(daily).map(function(d) {
    delete d.seen;
    return d;
  }).sort(function(a, b) { return a.date.localeCompare(b.date); });
}

function getFilteredRows(startDate, endDate) {
  var sheet = SpreadsheetApp.openById(getSpreadsheetId()).getSheetByName('RawData');
  var allData = sheet.getDataRange().getValues();
  var start = new Date(startDate);
  var end = new Date(endDate);
  end.setHours(23, 59, 59);

  return allData.slice(1).filter(function(row) {
    var d = new Date(row[0]);
    return d >= start && d <= end;
  });
}

function getSpreadsheetId() {
  return PropertiesService.getScriptProperties().getProperty('SPREADSHEET_ID');
}
