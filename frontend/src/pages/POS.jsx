import React,{useMemo,useState} from 'react';
import {Link,useNavigate,useOutletContext} from 'react-router-dom';
import {api,calculateLine,getError,money} from '../lib/api';
import {useLoad} from '../hooks/useLoad';
import {Alert,Button,Card,Empty,Field,Page,ProductAutocomplete,Select} from '../components/Ui';
import {EditorLines,useDocumentLines} from './Documents';
export default function POS(){
 const navigate=useNavigate();const {user}=useOutletContext();const {state,setState,clear,addProduct,updateLine,removeLine,totals}=useDocumentLines('POS',user.id);const {data:wh}=useLoad('warehouses',[]);
 const [method,setMethod]=useState('CASH');const [error,setError]=useState('');const [busy,setBusy]=useState(false);
 const warehouseId=state.warehouseId||(wh?.items||[]).find(x=>x.isDefault)?.id||0;
 const submit=async()=>{if(!state.lines.length){setError('Ajoutez des articles.');return};setBusy(true);setError('');try{const doc=await api('pos/sale',{method:'POST',body:{partyId:0,warehouseId:Number(warehouseId),method,lines:state.lines.map(l=>({productId:l.productId,qtyMilli:l.qtyMilli,unitPriceCents:l.unitPriceCents,discountBps:l.discountBps,vatBps:l.vatBps}))}});clear();navigate('/app/documents/'+doc.id)}catch(e){setError(getError(e))}finally{setBusy(false)}};
 return <Page title="Point de vente" description="Vente au comptoir — facture, encaissement et mouvement de stock dans une transaction MySQL"><Alert message={error}/><div className="pos-layout"><div className="stack"><EditorLines lines={state.lines} addProduct={addProduct} updateLine={updateLine} removeLine={removeLine} totals={totals}/></div><Card title="Encaissement"><div className="stack"><Field label="Emplacement"><Select value={warehouseId} onChange={e=>setState(s=>({...s,warehouseId:Number(e.target.value)}))} options={(wh?.items||[]).map(x=>({value:x.id,label:x.name}))}/></Field><Field label="Mode de règlement"><Select value={method} onChange={e=>setMethod(e.target.value)} options={[{value:'CASH',label:'Espèces'},{value:'CARD',label:'Carte bancaire'},{value:'TRANSFER',label:'Virement'}]}/></Field><div className="kpi"><div className="kpi-label">Total TTC</div><strong>{money(totals.totalCents)}</strong><small>Client : Au comptant</small></div><Button disabled={!state.lines.length||busy} variant="teal" onClick={submit}>Valider et encaisser</Button><p className="help-text">Si un article manque en stock, la transaction est annulée et aucun règlement n'est enregistré. Aucun stock n'est déduit deux fois.</p></div></Card></div></Page>;
}
