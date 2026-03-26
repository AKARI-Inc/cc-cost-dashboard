function writeAnalyticsToSheet(data, dateString) {
  var ss = SpreadsheetApp.openById(getSpreadsheetId());
  var sheet = ss.getSheetByName('RawData');

  // 冪等性: 同一日付の既存行を削除
  deleteDateRows(sheet, dateString);

  // フラット化してバルク書き込み
  var rows = [];
  var now = new Date();
  data.forEach(function(record) {
    var flatRows = flattenRecord(record, now);
    rows = rows.concat(flatRows);
  });

  if (rows.length > 0) {
    sheet.getRange(sheet.getLastRow() + 1, 1, rows.length, rows[0].length)
      .setValues(rows);
  }
}

function flattenRecord(record, fetchedAt) {
  var base = [
    record.date,
    record.actor.email_address || record.actor.api_key_name || '',
    record.actor.type,
    record.organization_id,
    record.customer_type,
    record.terminal_type || '',
    record.core_metrics.num_sessions,
    record.core_metrics.lines_of_code.added,
    record.core_metrics.lines_of_code.removed,
    record.core_metrics.commits_by_claude_code,
    record.core_metrics.pull_requests_by_claude_code,
    (record.tool_actions.edit_tool || {}).accepted || 0,
    (record.tool_actions.edit_tool || {}).rejected || 0,
    (record.tool_actions.multi_edit_tool || {}).accepted || 0,
    (record.tool_actions.multi_edit_tool || {}).rejected || 0,
    (record.tool_actions.write_tool || {}).accepted || 0,
    (record.tool_actions.write_tool || {}).rejected || 0,
    (record.tool_actions.notebook_edit_tool || {}).accepted || 0,
    (record.tool_actions.notebook_edit_tool || {}).rejected || 0,
  ];

  var models = record.model_breakdown || [];
  if (models.length === 0) {
    return [base.concat(['', 0, 0, 0, 0, 0, 'USD', fetchedAt])];
  }

  return models.map(function(m) {
    return base.concat([
      m.model,
      m.tokens.input,
      m.tokens.output,
      m.tokens.cache_read,
      m.tokens.cache_creation,
      m.estimated_cost.amount,
      m.estimated_cost.currency,
      fetchedAt
    ]);
  });
}

function deleteDateRows(sheet, dateString) {
  var data = sheet.getDataRange().getValues();
  for (var i = data.length - 1; i >= 1; i--) {
    var rowDate = Utilities.formatDate(new Date(data[i][0]), 'UTC', 'yyyy-MM-dd');
    if (rowDate === dateString) sheet.deleteRow(i + 1);
  }
}
