// HAUSV-672: the sidebar house title stays 18px serif without hyphenation
// (HAUSV-649), so a single long word such as "Janischhofweg" has to be fitted
// by stepping the font down (never below 13px) until the longest word fits its
// column; the CSS keeps overflow-wrap:anywhere as the last resort.
function fitHouseTitles(){
	var titles=document.querySelectorAll('aside.sidebar .house-header-copy strong');if(!titles.length)return;
	var canvas=fitHouseTitles.canvas||(fitHouseTitles.canvas=document.createElement('canvas'));
	var context=canvas.getContext('2d');if(!context)return;
	titles.forEach(function(title){
		title.style.fontSize='';
		var width=title.clientWidth;var words=(title.textContent||'').split(/[\s\u2010-]+/).filter(Boolean);
		if(!width||!words.length)return;
		var style=getComputedStyle(title);var size=Math.round(parseFloat(style.fontSize))||18;
		function longest(px){context.font=style.fontStyle+' '+style.fontWeight+' '+px+'px '+style.fontFamily;return Math.max.apply(null,words.map(function(word){return context.measureText(word).width;}));}
		function lines(){var range=document.createRange();range.selectNodeContents(title);return range.getClientRects().length;}
		var px=size;
		while(px>13&&(longest(px)>width||lines()>2)){px-=1;title.style.fontSize=px+'px';}
		if(px===size)title.style.fontSize='';
	});
}
document.addEventListener('DOMContentLoaded',function(){
	fitHouseTitles();
	if(document.fonts&&document.fonts.ready)document.fonts.ready.then(fitHouseTitles);
	var pending=0;window.addEventListener('resize',function(){cancelAnimationFrame(pending);pending=requestAnimationFrame(fitHouseTitles);});
});
document.addEventListener('DOMContentLoaded',function(){
	document.querySelectorAll('[data-house-picker]').forEach(function(picker){
		picker.querySelectorAll('[data-house-map-link]').forEach(function(link){link.addEventListener('click',function(event){event.stopPropagation();});});
		const search=picker.querySelector('[data-house-search]');
		const more=picker.querySelector('[data-house-more]');
		const empty=picker.querySelector('[data-house-empty]');
		const options=Array.from(picker.querySelectorAll('[data-house-option]'));
		function visibleButtons(){return options.filter(function(row){return !row.hidden;}).map(function(row){return row.querySelector('button:not([disabled])');}).filter(function(button){return button&&button.offsetParent!==null;});}
		if(search){search.addEventListener('input',function(){
			const query=search.value.trim().toLocaleLowerCase('de-AT');let matches=0;
			if(more&&query){more.open=true;}
			options.forEach(function(row){const show=!query||(row.dataset.houseSearchText||'').includes(query);row.hidden=!show;if(show)matches++;});
			if(empty)empty.hidden=matches>0;
		});}
		picker.addEventListener('keydown',function(event){
			if(event.key==='Escape'){event.preventDefault();picker.open=false;picker.querySelector(':scope > summary').focus();return;}
			if(event.key!=='ArrowDown'&&event.key!=='ArrowUp'&&!(event.key==='Enter'&&event.target===search))return;
			const buttons=visibleButtons();if(!buttons.length)return;event.preventDefault();
			const current=buttons.indexOf(document.activeElement);const step=event.key==='ArrowUp'?-1:1;buttons[(current+step+buttons.length)%buttons.length].focus();
		});
	});
});
