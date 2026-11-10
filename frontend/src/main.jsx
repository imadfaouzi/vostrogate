import React,{useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import {BrowserRouter,Navigate,Route,Routes} from 'react-router-dom';
import {api,setCSRF} from './lib/api';
import {Loading} from './components/Ui';
import Layout from './components/Layout';
import Setup from './pages/Setup';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import {PartiesList,PartyForm,PartyStatement} from './pages/Parties';
import {ProductList,ProductForm,Catalogue} from './pages/Products';
import {DocumentsList,DocumentEditor,DocumentDetail} from './pages/Documents';
import {StockPage,StockMovements} from './pages/Stock';
import {PaymentsPage,ChequePage} from './pages/Payments';
import Expenses from './pages/Expenses';
import Reports from './pages/Reports';
import Users from './pages/Users';
import Settings from './pages/Settings';
import POS from './pages/POS';
import './styles/app.css';
function App(){
 const [phase,setPhase]=useState('loading');const [user,setUser]=useState(null);
 const refresh=async()=>{const status=await api('setup/status');if(!status.installed){setPhase('setup');return};try{const p=await api('auth/me');setCSRF(p.csrf);setUser(p);setPhase('app')}catch{setPhase('login')}};
 useEffect(()=>{refresh().catch(()=>setPhase('offline'))},[]);
 const doLogin=async(username,password)=>{const response=await api('auth/login',{method:'POST',body:{username,password}});setCSRF(response.csrfToken);await refresh()};
 const doLogout=async()=>{try{await api('auth/logout',{method:'POST'})}finally{setCSRF('');setUser(null);setPhase('login')}};
 if(phase==='loading')return <div className="boot"><Loading/></div>;
 if(phase==='offline')return <div className="boot"><h2>API indisponible</h2><p>Vérifiez que le serveur Go et MySQL fonctionnent.</p><button onClick={refresh}>Réessayer</button></div>;
 if(phase==='setup')return <Setup onComplete={()=>setPhase('login')}/>;
 if(phase==='login')return <Login onLogin={doLogin}/>;
 return <BrowserRouter><Routes><Route path="/app" element={<Layout user={user} onLogout={doLogout}/>}>
  <Route index element={<Navigate replace to="dashboard"/>}/><Route path="dashboard" element={<Dashboard/>}/>
  <Route path="clients" element={<PartiesList kind="CUSTOMER"/>}/><Route path="clients/new" element={<PartyForm kind="CUSTOMER"/>}/><Route path="clients/:id" element={<PartyForm kind="CUSTOMER"/>}/><Route path="clients/:id/situation" element={<PartyStatement/>}/>
  <Route path="fournisseurs" element={<PartiesList kind="SUPPLIER"/>}/><Route path="fournisseurs/new" element={<PartyForm kind="SUPPLIER"/>}/><Route path="fournisseurs/:id" element={<PartyForm kind="SUPPLIER"/>}/><Route path="fournisseurs/:id/situation" element={<PartyStatement/>}/>
  <Route path="articles" element={<ProductList/>}/><Route path="articles/new" element={<ProductForm/>}/><Route path="articles/:id" element={<ProductForm/>}/><Route path="catalogue" element={<Catalogue/>}/>
  <Route path="achats" element={<Navigate replace to="/app/ventes/PURCHASE_RECEIPT"/>}/><Route path="ventes/:type" element={<DocumentsList/>}/><Route path="documents/new" element={<DocumentEditor/>}/><Route path="documents/:id" element={<DocumentDetail/>}/>
  <Route path="inventaire" element={<StockPage/>}/><Route path="inventaire/mouvements" element={<StockMovements/>}/>
  <Route path="paiements" element={<PaymentsPage/>}/><Route path="cheques" element={<ChequePage/>}/><Route path="depenses" element={<Expenses/>}/><Route path="rapports" element={<Reports/>}/><Route path="pos" element={<POS/>}/>
  <Route path="utilisateurs" element={<Users/>}/><Route path="parametres" element={<Settings/>}/><Route path="audit" element={<Reports audit/>}/>
 </Route><Route path="*" element={<Navigate replace to="/app/dashboard"/>}/></Routes></BrowserRouter>;
}
createRoot(document.getElementById('root')).render(<App/>);
