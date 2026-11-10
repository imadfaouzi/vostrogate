import React,{useEffect,useRef,useState} from 'react';
import {api,qs} from '../lib/api';
import {Icon} from './Ui';
export default function PartyAutocomplete({kind='CUSTOMER',selected,onSelect}){
 const [q,setQ]=useState('');const [list,setList]=useState([]);const [open,setOpen]=useState(false);const ref=useRef(null);
 useEffect(()=>{const close=e=>{if(!ref.current?.contains(e.target))setOpen(false)};document.addEventListener('mousedown',close);return()=>document.removeEventListener('mousedown',close)},[]);
 useEffect(()=>{if(selected||q.trim().length<1){setList([]);return};let live=true;const timer=setTimeout(()=>{api('parties?'+qs({kind,q,limit:15})).then(x=>{if(live){setList(x.items||[]);setOpen(true)}}).catch(()=>{})},160);return()=>{live=false;clearTimeout(timer)}},[q,kind,selected]);
 return <div ref={ref} className="autocomplete">{selected?<div className="searchfield"><Icon name="check"/><strong>{selected.name} ({selected.code})</strong><button type="button" className="icon-button" title="Changer" onClick={()=>{onSelect(null);setQ('');setOpen(false)}}>×</button></div>:<><div className="searchfield product-search"><Icon name="search"/><input placeholder={kind==='SUPPLIER'?'Nom ou code fournisseur...':'Nom ou code client...'} value={q} onChange={e=>setQ(e.target.value)} onFocus={()=>setOpen(true)}/></div>{open&&q&&<div className="suggestions">{list.length?list.map(p=><button className="suggestion" type="button" key={p.id} onClick={()=>{onSelect(p);setOpen(false);setQ('')}}><span><strong>{p.name}</strong><small>{p.code} · {p.phone}</small></span><span>{p.city}</span></button>):<div className="noresults">Aucun résultat. Ajoutez le contact depuis son module.</div>}</div>}</> }</div>;
}
