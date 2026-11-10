import React,{useState} from 'react';
import {Alert,Button,Field,Input} from '../components/Ui';
export default function Login({onLogin}){
 const [username,setUsername]=useState('');const [password,setPassword]=useState('');const [error,setError]=useState('');const [busy,setBusy]=useState(false);
 const login=async(e)=>{e.preventDefault();setBusy(true);setError('');try{await onLogin(username,password)}catch(e){setError(e.message)}finally{setBusy(false)}};
 return <div className="login-page"><div className="login-card"><img className="login-logo" src="/brand/logo.svg" alt="Logo Vostrogate"/><h1>Connexion</h1><p className="muted">Vostrogate 1.0.0 · ERP commercial</p><Alert message={error}/><form onSubmit={login} className="stack"><Field label="Identifiant ou adresse e-mail"><Input autoFocus required value={username} onChange={e=>setUsername(e.target.value)} autoComplete="username"/></Field><Field label="Mot de passe"><Input required type="password" value={password} onChange={e=>setPassword(e.target.value)} autoComplete="current-password"/></Field><Button variant="primary" disabled={busy} type="submit">{busy?'Connexion...':'Se connecter →'}</Button></form><small className="muted">Authentification sécurisée par session serveur.</small></div></div>
}
