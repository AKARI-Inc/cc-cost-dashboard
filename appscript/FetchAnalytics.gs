var API_BASE = 'https://api.anthropic.com/v1/organizations/usage_report/claude_code';
var MAX_RETRIES = 3;

function fetchDailyAnalytics(dateString) {
  var allData = [];
  var page = null;

  do {
    var url = API_BASE + '?starting_at=' + dateString + '&limit=1000';
    if (page) url += '&page=' + encodeURIComponent(page);

    var result = callAnthropicAPI(url);
    allData = allData.concat(result.data);
    page = result.next_page;
  } while (result.has_more === true);

  return allData;
}

function callAnthropicAPI(url) {
  var apiKey = PropertiesService.getScriptProperties().getProperty('ADMIN_API_KEY');
  var options = {
    method: 'get',
    headers: {
      'anthropic-version': '2023-06-01',
      'x-api-key': apiKey,
      'User-Agent': 'CCDashboard/1.0.0'
    },
    muteHttpExceptions: true
  };

  for (var attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    try {
      var response = UrlFetchApp.fetch(url, options);
      var code = response.getResponseCode();

      if (code === 200) return JSON.parse(response.getContentText());
      if (code === 429) { Utilities.sleep(5000 * attempt); continue; }
      if (code >= 500) { Utilities.sleep(2000 * attempt); continue; }
      if (code === 401 || code === 403) throw new Error('Invalid API key (HTTP ' + code + ')');
      throw new Error('API error: HTTP ' + code);
    } catch (e) {
      if (attempt === MAX_RETRIES) throw e;
      Utilities.sleep(2000);
    }
  }
}
