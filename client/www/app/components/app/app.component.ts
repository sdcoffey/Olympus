import {Component} from 'angular2/core';
import {NodeListComponent} from '../nodelist/nodelist.component';
import {ApiClient} from '../../services/apiclient';
import {HTTP_PROVIDERS} from 'angular2/http';
import {RouteConfig, Router, ROUTER_DIRECTIVES, ROUTER_PROVIDERS} from 'angular2/router';

@Component({
  selector: 'app',
  templateUrl: 'app/components/app/app.html',
  styleUrls: ['app/components/app/app.css'],
  directives: [NodeListComponent, ROUTER_DIRECTIVES],
  providers: [ApiClient, HTTP_PROVIDERS, ROUTER_PROVIDERS]
})
@RouteConfig([
  {
    path: '/browse/:parentId',
    name: 'Browse',
    component: NodeListComponent
  },
  {
    path: '/',
    redirectTo: ['Browse', {parentId: 'rootNode'}],
    useAsDefault: true
  }
])
export class AppComponent {}
