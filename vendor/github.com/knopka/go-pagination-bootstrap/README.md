#Usage

```
pager := pagination.New(TotalCount, PerPage, CurrentPage, strURI)
```

for example

```
pager := New(100, 10, 2, "http://www.abc.com/news?param1=abc&param2=123&page=2")
```

in html template

```
{{ .pager.Render }}
```

then it will generate bootstrap friendly pagination html elements, such as


```
<ul class="pagination pagination-sm">
  <li><a href="?page=5">&lt;</a></li>
  
    <li><a href="?page=1">1</a></li>
  
    <li><a href="?page=2">2</a></li>
  
    <li class="disabled"><span>...</span></li>
	
	  <li><a href="?page=3">3</a></li>
	
	  <li><a href="?page=4">4</a></li>
	
	  <li><a href="?page=5">5</a></li>
	
	  <li class="active"><span>6</span></li>
	
	  <li><a href="?page=7">7</a></li>
	
	  <li><a href="?page=8">8</a></li>
	
	  <li><a href="?page=9">9</a></li>  
  
	<li class="disabled"><span>...</span></li>
	
	  <li><a href="?page=13">13</a></li>
	
	  <li><a href="?page=14">14</a></li>
	
  
  <li><a href="?page=7">&gt;</a></li>
</ul>
```