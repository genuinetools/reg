// pretty date function
function prettyDate(time){
    var date = new Date((time || "").replace(/-/g,"/").replace(/[TZ]/g," ")),
    diff = (((new Date()).getTime() - date.getTime()) / 1000),
    day_diff = Math.floor(diff / 86400);

    if (isNaN(day_diff) || day_diff < 0)
        return;

    return day_diff == 0 && (
            diff < 60 && "just now" ||
            diff < 120 && "1 minute ago" ||
            diff < 3600 && Math.floor( diff / 60 ) + " minutes ago" ||
            diff < 7200 && "1 hour ago" ||
            diff < 86400 && Math.floor( diff / 3600 ) + " hours ago") ||
        day_diff == 1 && "Yesterday" ||
        day_diff < 7 && day_diff + " days ago" ||
        day_diff < 31 && Math.ceil( day_diff / 7 ) + " weeks ago" ||
        day_diff > 31 && Math.round(day_diff / 31) + " months ago";
}

// search function
function search(search_val){
    var suche = search_val.toLowerCase();
    var table = document.getElementById("directory");
    var cellNr = 1;
    var ele;
    for (var r = 1; r < table.rows.length; r++){
        ele = table.rows[r].cells[cellNr].innerHTML.replace(/<[^>]+>/g,"");
        if (ele.toLowerCase().indexOf(suche)>=0 ) {
            table.rows[r].style.display = '';
        } else {
            table.rows[r].style.display = 'none';
        }
    }
}

const allVulns = {};
function updateVulnerabilityElement(report) {
  const id = report.Repo + ':' + report.Tag;
  const element = document.getElementById(id);

  if (element) {
    element.innerHTML = report.BadVulns;
  } else {
    console.log("element not found for given id ", id);
  }
}
function processFromCache(url, urlMatch) {
  const baseUrl = urlMatch[1];
  const repo = urlMatch[2];
  const tag = urlMatch[3];
  // Format matches our expectations. Now we need to check our cache
  // Our cache is a state machine:
  // allVulns[baseUrl] === undefined // empty cache, nothing has happened
  // allVulns[baseUrl] === 'pending' // XHR sent, not yet received
  // allVulns[baseUrl] === 'nodata'  // XHR sent, received and 404 or other error occurred
  // allVulns[baseUrl] === {}        // We're all good, use the cache instead
  const state = allVulns[baseUrl];
  switch (state) {
    case undefined: // next state, 'pending'
      allVulns[baseUrl] = 'pending';
      const xhr = new XMLHttpRequest();
      xhr.open('GET', baseUrl + '/vulns.json');
      xhr.onload = () => {
        // xhr complete, next state 'nodata', or data
        if (xhr.status === 200) {
          try {
            const serverVulns = JSON.parse(xhr.responseText);
            allVulns[baseUrl] = {};
            serverVulns.forEach(vuln => {allVulns[baseUrl][vuln.Repo + ':' + vuln.Tag] = vuln;});
          }catch (err) {
            allVulns[baseUrl] = 'nodata';
            console.error(err);
          }
          window.setTimeout(() => loadVulnerabilityCount(url), 0);
        }else {
          allVulns[baseUrl] = 'nodata';
          window.setTimeout(() => loadVulnerabilityCount(url), 0);
        }
      };
      xhr.send();
      break;
    case 'pending': // next state, pending
      window.setTimeout(() => loadVulnerabilityCount(url), 0);
      break;
    case 'nodata': // terminal state, call failed
      return false; // No data, caller must handle outside of cache
    default: // terminal state
      updateVulnerabilityElement(allVulns[baseUrl][repo + ':' + tag])
      break;
  }
  return true; // Let caller know we have control
}
function loadVulnerabilityCount(url){
  // url: /repo/<repo>/tag/<tag>/vulns.json
  const urlMatch = url.match(/^(\/repo\/(\w*))\/tag\/([^\/]+)\/vulns\.json$/)
  if (urlMatch && processFromCache(url, urlMatch)) {
    return;
  }

  // Cannot process from cache/parent repo level. Fallback to existing behavior
  var xhr = new XMLHttpRequest();
  xhr.open('GET', url);
  xhr.onload = () => {
    if (xhr.status === 200) {
      const report = JSON.parse(xhr.responseText);
      updateVulnerabilityElement(report)
    }
  };
  xhr.send();
}

function summarizeMultiArchImages(){
  const rows=document.querySelectorAll('table tr');
  const allcells=Array.from(rows.entries()).map(r => {
    return {
      innerText : r[1].childNodes[3].innerText,
      row: r[1],
      cell: r[1].childNodes[3]
    }
  });

  rows.forEach((r, i) => {
    const tagCell = r.childNodes[3];
    const tag = tagCell.innerText;
    // The rest of this code is specific to how my tags are named...
    if (!tag || tag.match('_')) { return; }
    const re = new RegExp('^' + tag + '__(\\w*)_(\\w*)$');
    const matchedCells = allcells.map
                            (cell => {
                              const match = cell.innerText.match(re);
                              if (!match) return null;
                              return {
                                match,
                                cell
                              };
                            }).filter(c => !!c).sort(
                              (a,b) => a.match[1] + a.match[2] > b.match[1] + b.match[2]
                            );
    matchedCells.forEach(c => {
      const newnode = c.cell.cell.childNodes[1].cloneNode();
      newnode.innerText = c.match[1] + '/' + c.match[2];
      newnode.className = 'arch-variant';
      tagCell.childNodes[1].className = 'tag';
      tagCell.appendChild(newnode);
      c.cell.row.remove();
    });
  });
}

var el = document.querySelectorAll('tr:nth-child(2)')[0].querySelectorAll('td:nth-child(2)')[0];
if (el.textContent == 'Parent Directory'){
    var parent_row = document.querySelectorAll('tr:nth-child(2)')[0];
    if (parent_row.classList){
        parent_row.classList.add('parent');
    } else {
        parent_row.className += ' ' + 'parent';
    }
}

// Tag page - adjust multi-arch images into pretty format
var el = document.querySelectorAll('tr th:nth-child(2)')[0];
if (el.textContent === 'Tag') {
  summarizeMultiArchImages();
}

// Adjust links from server
var cells = document.querySelectorAll('td a');
Array.prototype.forEach.call(cells, function(item, index){
    var link = item.getAttribute('href');
    link = link.replace('.html', '');
    item.setAttribute('href', link);
});

var our_table = document.querySelectorAll('table')[0];
our_table.setAttribute('id', 'directory');

// search script
var search_input = document.querySelectorAll('input[name="filter"]')[0];
var clear_button = document.querySelectorAll('a.clear')[0];

if (search_input) {
  if (search_input.value !== ''){
      search(search_input.value);
  }

  search_input.addEventListener('keyup', function(e){
      e.preventDefault();
      search(search_input.value);
  });

  search_input.addEventListener('keypress', function(e){
      if ( e.which == 13 ) {
          e.preventDefault();
      }
  });
}

if (clear_button) {
  clear_button.addEventListener('click', function(e){
      search_input.value = '';
      search('');
  });
}
