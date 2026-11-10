import React,{useEffect,useState} from 'react';
import {Link,useNavigate,useParams,useOutletContext} from 'react-router-dom';
import {api,getError,money,parseMoney,parseQty,qs,quantity} from '../lib/api';
import {useDebounced,useLoad} from '../hooks/useLoad';
import useDraft from '../hooks/useDraft';
import {ActionLink,Alert,Button,Card,CsvButton,Empty,Field,FormActions,Input,Loading,Page,Pager,Select,Textarea,Toolbar} from '../components/Ui';
const blank={sku:'',barcode:'',name:'',description:'',categoryId:0,brandId:0,unitId:0,purchasePriceCents:0,salePriceCents:0,vatBps:2000,minStockMilli:0,initialStockMilli:0,warehouseId:0,isActive:true};
export function ProductList(){
 const [q,setQ]=useState('');const [offset,setOffset]=useState(0);const searched=useDebounced(q);const {data,loading,error}=useLoad('products?'+qs({q:searched,offset,limit:50}),[searched,offset]);
 const rows=data?.items||[];
 return <Page title="Articles" description="Catalogue, références, codes-barres, prix et stock" actions={<ActionLink to="/app/articles/new" icon="plus">Nouvel article</ActionLink>}><Card><Toolbar q={q} setQ={v=>{setQ(v);setOffset(0)}} placeholder="Rechercher par nom, référence ou code-barres" other={<><Link className="btn" to="/app/catalogue">Référentiels</Link><CsvButton rows={rows} filename="articles.csv"/></>}/><Alert message={error}/>{loading?<Loading/>:rows.length?<><div className="table-wrap"><table className="table"><thead><tr><th>Référence</th><th>Code-barres</th><th>Article</th><th>Prix achat</th><th>Prix vente</th><th>Bénéfice potentiel</th><th>Stock</th><th>Actions</th></tr></thead><tbody>{rows.map(p=><tr key={p.id}><td><strong>{p.sku}</strong></td><td>{p.barcode||'—'}</td><td>{p.name}</td><td>{money(p.purchasePriceCents)}</td><td>{money(p.salePriceCents)}</td><td>{money(p.salePriceCents-p.purchasePriceCents)}</td><td><span className={p.stockMilli<p.minStockMilli?'amount-danger':'amount-good'}>{quantity(p.stockMilli)}</span></td><td><Link className="btn small" to={'/app/articles/'+p.id}>Modifier</Link></td></tr>)}</tbody></table></div><Pager offset={offset} setOffset={setOffset} total={data?.total||0}/></>:<Empty message="Aucun article. Ajoutez une référence au catalogue."/>}</Card></Page>;
}
export function ProductForm(){
 const {id}=useParams();const {user}=useOutletContext();const navigate=useNavigate();const [form,setForm,clear]=useDraft('u'+user.id+'-product-'+(id||'new'),blank);const [error,setError]=useState('');const [busy,setBusy]=useState(false);
 const {data:refs}=useLoad('catalog',[]);const {data:warehouses}=useLoad('warehouses',[]);
 useEffect(()=>{if(id){api('products/'+id).then(setForm).catch(e=>setError(e.message))}},[id]);
 const update=(k,v)=>setForm(s=>({...s,[k]:v}));
 const refOptions=kind=>[{value:0,label:'— Aucun —'},...(refs?.items||[]).filter(x=>x.kind===kind).map(x=>({value:x.id,label:x.name}))];
 const save=async(e)=>{e.preventDefault();setBusy(true);setError('');try{const res=await api('products'+(id?'/'+id:''),{method:id?'PUT':'POST',body:form});clear();navigate('/app/articles/'+(res.id||id))}catch(e){setError(getError(e))}finally{setBusy(false)}};
 return <Page title={id?'Modifier un article':'Nouvel article'} description="Référence, code-barres, tarifs et stock initial"><Card><Alert message={error}/><form onSubmit={save}><div className="form-grid three">
 <Field label="Référence article" required><Input required value={form.sku} onChange={e=>update('sku',e.target.value)}/></Field>
 <Field label="Code-barres (EAN / UPC)"><Input value={form.barcode||''} onChange={e=>update('barcode',e.target.value)} placeholder="Scan possible avec lecteur USB"/></Field>
 <Field label="Désignation de l'article" required><Input required value={form.name} onChange={e=>update('name',e.target.value)}/></Field>
 <Field label="Catégorie"><Select value={form.categoryId||0} options={refOptions('category')} onChange={e=>update('categoryId',Number(e.target.value))}/></Field>
 <Field label="Marque"><Select value={form.brandId||0} options={refOptions('brand')} onChange={e=>update('brandId',Number(e.target.value))}/></Field>
 <Field label="Unité"><Select value={form.unitId||0} options={refOptions('unit')} onChange={e=>update('unitId',Number(e.target.value))}/></Field>
 <Field label="Prix d'achat (DH)"><Input type="number" min="0" step="0.01" value={(form.purchasePriceCents||0)/100} onChange={e=>update('purchasePriceCents',parseMoney(e.target.value))}/></Field>
 <Field label="Prix de vente (DH)"><Input type="number" min="0" step="0.01" value={(form.salePriceCents||0)/100} onChange={e=>update('salePriceCents',parseMoney(e.target.value))}/></Field>
 <Field label="TVA (%)"><Input type="number" min="0" max="100" step="0.01" value={(form.vatBps||0)/100} onChange={e=>update('vatBps',Math.round(Number(e.target.value)*100))}/></Field>
 <Field label="Seuil d'alerte stock"><Input type="number" step="0.001" min="0" value={(form.minStockMilli||0)/1000} onChange={e=>update('minStockMilli',parseQty(e.target.value))}/></Field>
 {!id&&<Field label="Stock initial"><Input type="number" step="0.001" min="0" value={(form.initialStockMilli||0)/1000} onChange={e=>update('initialStockMilli',parseQty(e.target.value))}/></Field>}
 {!id&&<Field label="Dépôt initial"><Select value={form.warehouseId||0} onChange={e=>update('warehouseId',Number(e.target.value))} options={[{value:0,label:'Dépôt principal'},...(warehouses?.items||[]).map(x=>({value:x.id,label:x.name}))]}/></Field>}
 <Field label="Description" ><Textarea value={form.description||''} onChange={e=>update('description',e.target.value)}/></Field>
 </div><FormActions busy={busy} cancel={()=>navigate('/app/articles')}/></form></Card></Page>;
}
const kinds=[['category','Catégories'],['brand','Marques'],['unit','Unités'],['price_group','Groupes de prix'],['expense_category','Catégories de dépense']];
export function Catalogue(){
 const [kind,setKind]=useState('category');const [name,setName]=useState('');const [error,setError]=useState('');const [tick,setTick]=useState(0);const {data,loading}=useLoad('catalog',[tick]);
 const save=async(e)=>{e.preventDefault();setError('');try{await api('catalog',{method:'POST',body:{kind,name}});setName('');setTick(t=>t+1)}catch(e){setError(e.message)}};
 return <Page title="Référentiels d'articles" description="Catégories, unités, marques et groupes de prix"><div className="grid grid-2"><Card title="Liste des valeurs"><div className="pill-switch">{kinds.map(([k,label])=><button key={k} className={kind===k?'active':''} onClick={()=>setKind(k)}>{label}</button>)}</div>{loading?<Loading/>:<div className="table-wrap" style={{marginTop:15}}><table className="table"><thead><tr><th>Nom</th><th>Statut</th></tr></thead><tbody>{(data?.items||[]).filter(x=>x.kind===kind).map(x=><tr key={x.id}><td>{x.name}</td><td>{x.isActive?'Actif':'Inactif'}</td></tr>)}</tbody></table></div>}</Card><Card title="Ajouter une valeur"><Alert message={error}/><form onSubmit={save}><div className="stack"><Field label="Type"><Select value={kind} onChange={e=>setKind(e.target.value)} options={kinds.map(([value,label])=>({value,label}))}/></Field><Field label="Libellé" required><Input required value={name} onChange={e=>setName(e.target.value)}/></Field><Button variant="primary" type="submit">+ Ajouter</Button></div></form></Card></div></Page>;
}
