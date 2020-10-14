/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.citrusframework.yaks.camelk;

import com.consol.citrus.Citrus;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;
import io.fabric8.kubernetes.client.dsl.base.CustomResourceDefinitionContext;

/**
 * @author Christoph Deppisch
 */
public class CamelKSupport {

    public static final String CAMELK_CRD_GROUP = "camel.apache.org";

    private CamelKSupport() {
        // prevent instantiation of utility class
    }

    public static KubernetesClient getKubernetesClient(Citrus citrus) {
        if (citrus.getCitrusContext().getReferenceResolver().resolveAll(KubernetesClient.class).size() == 1L) {
            return citrus.getCitrusContext().getReferenceResolver().resolve(KubernetesClient.class);
        } else {
            return new DefaultKubernetesClient();
        }
    }

    public static CustomResourceDefinitionContext integrationCRDContext(String version) {
        return camelkCRDContext("integrations", version);
    }

    public static CustomResourceDefinitionContext camelkCRDContext(String kind, String version) {
        return new CustomResourceDefinitionContext.Builder()
                .withName(kind + "." + CAMELK_CRD_GROUP)
                .withGroup(CAMELK_CRD_GROUP)
                .withVersion(version)
                .withPlural(kind)
                .withScope("Namespaced")
                .build();
    }
}
