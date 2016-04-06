import {Component} from 'angular2/core';
import {FileListComponent} from '../filelist/filelist.component';
import {ApiClient} from '../../services/apiclient';
import {HTTP_PROVIDERS} from 'angular2/http';
import {RouteConfig, Router, ROUTER_DIRECTIVES, ROUTER_PROVIDERS} from 'angular2/router';

@Component({
  selector: 'app',
  templateUrl: 'app/components/app/app.html',
  styleUrls: ['app/components/app/app.css'],
  directives: [FileListComponent, ROUTER_DIRECTIVES],
  providers: [ApiClient, HTTP_PROVIDERS, ROUTER_PROVIDERS]
})
@RouteConfig([
  {
    path: '/browse/:parentId',
    name: 'Browse',
    component: FileListComponent
  },
  {
    path: '/',
    redirectTo: ['Browse', {parentId: 'rootNode'}],
    useAsDefault: true
  }
])
export class AppComponent {}
