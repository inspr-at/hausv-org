// HAUSV-672/673: the sidebar house title stays 18px serif without hyphenation
// (HAUSV-649). When a word would have to break inside itself, or the title
// needs more than two lines, the font steps down (never below 13px). The check
// reads the rendered line boxes, not a canvas estimate, so it holds for every
// font the browser actually uses.
function fitHouseTitles(){
	var titles=document.querySelectorAll('aside.sidebar .house-header-copy strong');if(!titles.length)return;
	function textNodes(node){var out=[],walker=document.createTreeWalker(node,NodeFilter.SHOW_TEXT);while(walker.nextNode())out.push(walker.currentNode);return out;}
	function brokenWord(title){
		var nodes=textNodes(title);
		for(var i=0;i<nodes.length;i++){var text=nodes[i].data,re=/[^\s\u2010-]+/g,match;
			while((match=re.exec(text))){var range=document.createRange();range.setStart(nodes[i],match.index);range.setEnd(nodes[i],match.index+match[0].length);if(range.getClientRects().length>1)return true;}}
		return false;
	}
	function lines(title){var range=document.createRange();range.selectNodeContents(title);return range.getClientRects().length;}
	titles.forEach(function(title){
		title.style.fontSize='';
		if(!title.clientWidth||!(title.textContent||'').trim())return;
		var size=Math.round(parseFloat(getComputedStyle(title).fontSize))||18,px=size;
		while(px>13&&(brokenWord(title)||lines(title)>2)){px-=1;title.style.fontSize=px+'px';}
		if(px===size)title.style.fontSize='';
	});
}
document.addEventListener('DOMContentLoaded',function(){
	fitHouseTitles();
	if(document.fonts){if(document.fonts.ready)document.fonts.ready.then(fitHouseTitles);if(document.fonts.addEventListener)document.fonts.addEventListener('loadingdone',fitHouseTitles);}
	window.addEventListener('load',fitHouseTitles);
	var pending=0;window.addEventListener('resize',function(){cancelAnimationFrame(pending);pending=requestAnimationFrame(fitHouseTitles);});
});
document.addEventListener('DOMContentLoaded',function(){
	document.querySelectorAll('[data-house-picker]').forEach(function(picker){
		picker.querySelectorAll('[data-house-map-link]').forEach(function(link){link.addEventListener('click',function(event){event.stopPropagation();});});
		const search=picker.querySelector('[data-house-search]');
		const more=picker.querySelector('[data-house-more]');
		const empty=picker.querySelector('[data-house-empty]');
		const options=Array.from(picker.querySelectorAll('[data-house-option]'));
		let wasSearching=false,moreWasOpen=false;
		function visibleButtons(){return options.filter(function(row){return !row.hidden;}).map(function(row){return row.querySelector('button:not([disabled])');}).filter(function(button){return button&&button.offsetParent!==null;});}
		if(search){search.addEventListener('input',function(){
			const query=search.value.trim().toLocaleLowerCase('de-AT');let matches=0;
			if(more){
				if(query&&!wasSearching)moreWasOpen=more.open;
				more.open=query?true:moreWasOpen;
			}
			wasSearching=!!query;
			options.forEach(function(row){const show=!query||(row.dataset.houseSearchText||'').includes(query);row.hidden=!show;if(show)matches++;});
			if(more)more.hidden=!!query&&!options.some(function(row){return more.contains(row)&&!row.hidden;});
			if(empty)empty.hidden=matches>0;
		});}
		picker.addEventListener('keydown',function(event){
			if(event.key==='Escape'){event.preventDefault();picker.open=false;picker.querySelector(':scope > summary').focus();return;}
			if(event.key!=='ArrowDown'&&event.key!=='ArrowUp'&&!(event.key==='Enter'&&event.target===search))return;
			const buttons=visibleButtons();if(!buttons.length)return;event.preventDefault();
			const current=buttons.indexOf(document.activeElement);const step=event.key==='ArrowUp'?-1:1;const next=current<0?(step<0?buttons.length-1:0):(current+step+buttons.length)%buttons.length;buttons[next].focus();
		});
	});
});
