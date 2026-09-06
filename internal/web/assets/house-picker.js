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
